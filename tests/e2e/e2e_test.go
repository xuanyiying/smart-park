//go:build integration

// Package e2e_test wires the vehicle, billing and payment services in-process
// with real PostgreSQL/Redis and real HTTP/gRPC servers, then drives the full
// parking flow over HTTP exactly as a lane camera and the mini program would:
//
//	入场 → 出场计费(跨服务 gRPC) → 创建支付单(服务端权威金额) → 结算回写 → 无感放行
//
// Run with: INTEGRATION=1 go test -tags=integration ./tests/e2e
package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	kratosgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pkgmiddleware "github.com/xuanyiying/smart-park/pkg/middleware"

	billingv1 "github.com/xuanyiying/smart-park/api/billing/v1"
	paymentv1 "github.com/xuanyiying/smart-park/api/payment/v1"
	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	billingbiz "github.com/xuanyiying/smart-park/internal/billing/biz"
	billingdata "github.com/xuanyiying/smart-park/internal/billing/data"
	billingent "github.com/xuanyiying/smart-park/internal/billing/data/ent"
	billingservice "github.com/xuanyiying/smart-park/internal/billing/service"
	paymentbiz "github.com/xuanyiying/smart-park/internal/payment/biz"
	paymentdata "github.com/xuanyiying/smart-park/internal/payment/data"
	paymentent "github.com/xuanyiying/smart-park/internal/payment/data/ent"
	paymentservice "github.com/xuanyiying/smart-park/internal/payment/service"
	vehiclebiz "github.com/xuanyiying/smart-park/internal/vehicle/biz"
	vbilling "github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
	vehicledata "github.com/xuanyiying/smart-park/internal/vehicle/data"
	vehicleent "github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	vehicleservice "github.com/xuanyiying/smart-park/internal/vehicle/service"
	"github.com/xuanyiying/smart-park/pkg/lock"
	"github.com/xuanyiying/smart-park/pkg/multitenancy"
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func testDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		envOr("DB_HOST", "127.0.0.1"), envOr("DB_PORT", "5432"),
		envOr("DB_USER", "postgres"), envOr("DB_PASSWORD", "postgres"),
		envOr("DB_NAME", "parking_test"))
}

func testLogger() log.Logger { return log.NewStdLogger(io.Discard) }

// randomPlate 生成符合新能源车牌格式的唯一测试车牌：
// 省份简称 + 发牌机关字母 + "D"（纯电动标识）+ 5 位序号（不含 I/O），
// 例如 京AD3F7K2。
func randomPlate(regionPrefix string) string {
	const sequenceCharset = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	sequence := make([]byte, 5)
	for i := range sequence {
		sequence[i] = sequenceCharset[rand.Intn(len(sequenceCharset))]
	}
	return regionPrefix + "D" + string(sequence)
}

type e2eEnv struct {
	vehicleHTTP string // http://127.0.0.1:port
	paymentHTTP string
	vehicleRepo vehiclebiz.VehicleRepo
	orderRepo   paymentbiz.OrderRepo
	tenant      *multitenancy.TenantInfo
}

func mustListener(t *testing.T) net.Listener {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	return lis
}

// startServer runs a kratos server in the background; a start failure fails the test.
func startServer(t *testing.T, name string, start func(context.Context) error) {
	t.Helper()
	go func() {
		if err := start(context.Background()); err != nil {
			t.Errorf("%s server exited: %v", name, err)
		}
	}()
}

func waitReady(t *testing.T, lis net.Listener) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", lis.Addr().String(), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server on %s not ready", lis.Addr())
}

type e2eOpts struct {
	billingDown bool // 模拟 billing 服务不可达（出场必须不放行）
}

func setupE2E(t *testing.T, apply ...func(*e2eOpts)) *e2eEnv {
	t.Helper()
	opts := &e2eOpts{}
	for _, f := range apply {
		f(opts)
	}
	ctx := context.Background()
	logger := testLogger()
	env := &e2eEnv{}

	// billing gRPC 地址：正常时启动真实服务；注入故障时指向一个已关闭的端口，
	// gRPC 懒连接在真正调用 CalculateFee 时才会报 connection refused。
	var billingAddr string
	if !opts.billingDown {
		// ---------- billing：真实 gRPC server，供 vehicle 跨服务调用 ----------
		bclient, err := billingent.Open("postgres", testDSN())
		require.NoError(t, err)
		t.Cleanup(func() { _ = bclient.Close() })
		require.NoError(t, bclient.Schema.Create(ctx))

		bdata, bcleanup, err := billingdata.NewData(bclient, logger)
		require.NoError(t, err)
		t.Cleanup(bcleanup)

		billingUC := billingbiz.NewBillingUseCase(billingdata.NewBillingRuleRepo(bdata), logger, nil)
		billingSvc := billingservice.NewBillingService(billingUC, logger)

		blis := mustListener(t)
		bgs := kratosgrpc.NewServer(kratosgrpc.Listener(blis))
		billingv1.RegisterBillingServiceServer(bgs, billingSvc)
		startServer(t, "billing-grpc", bgs.Start)
		waitReady(t, blis)
		billingAddr = blis.Addr().String()
	} else {
		dead, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		billingAddr = dead.Addr().String()
		_ = dead.Close() // 立即关闭，制造 connection refused
	}

	// ---------- vehicle ----------
	vclient, err := vehicleent.Open("postgres", testDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = vclient.Close() })
	require.NoError(t, vclient.Schema.Create(ctx))

	// 清空流程相关表，保证种子数据确定性（测试库专用）
	_, _ = vclient.ParkingRecord.Delete().Exec(ctx)
	_, _ = vclient.Device.Delete().Exec(ctx)
	_, _ = vclient.Lane.Delete().Exec(ctx)
	_, _ = vclient.Vehicle.Delete().Exec(ctx)

	vdata, vcleanup, err := vehicledata.NewData(vclient, testDSN(), logger)
	require.NoError(t, err)
	t.Cleanup(vcleanup)
	vrepo := vehicledata.NewVehicleRepo(vdata)
	env.vehicleRepo = vrepo

	// 种子数据：入口/出口车道 + 相机（带租户，模拟生产多租户写入）
	tenantID := uuid.New()
	env.tenant = &multitenancy.TenantInfo{ID: tenantID}
	lotID := uuid.New()
	laneEntry, laneExit := uuid.New(), uuid.New()
	require.NoError(t, vclient.Lane.Create().
		SetID(laneEntry).SetLaneNo(1).SetLotID(lotID).
		SetDirection("entry").SetTenantID(tenantID).Exec(ctx))
	require.NoError(t, vclient.Lane.Create().
		SetID(laneExit).SetLaneNo(2).SetLotID(lotID).
		SetDirection("exit").SetTenantID(tenantID).Exec(ctx))

	seedDevice := func(code string, laneID uuid.UUID) {
		require.NoError(t, vclient.Device.Create().
			SetDeviceID(code).SetDeviceSecret("dev_"+uuid.New().String()).
			SetDeviceType("camera").SetStatus("active").
			SetLaneID(laneID).SetLotID(lotID).SetTenantID(tenantID).
			SetLastHeartbeat(time.Now()).Exec(ctx))
	}
	seedDevice("CAM001", laneEntry)
	seedDevice("CAM003", laneExit)

	redisClient := redis.NewClient(&redis.Options{Addr: envOr("REDIS_ADDR", "127.0.0.1:6379")})
	require.NoError(t, redisClient.Ping(ctx).Err())
	t.Cleanup(func() { _ = redisClient.Close() })

	billingConn, err := kratosgrpc.DialInsecure(ctx, kratosgrpc.WithEndpoint(billingAddr))
	require.NoError(t, err)
	t.Cleanup(func() { _ = billingConn.Close() })

	entryExitUC := vehiclebiz.NewEntryExitUseCase(
		vrepo,
		vbilling.NewClient(billingv1.NewBillingServiceClient(billingConn), logger),
		nil, // MQTT：开闸命令失败不阻塞流程（生产由 main 装配真实客户端）
		lock.NewRedisLockRepo(redisClient, logger, "e2e-vehicle"),
		nil, nil, logger)

	vehicleSvc := vehicleservice.NewVehicleService(
		entryExitUC,
		vehiclebiz.NewDeviceUseCase(vrepo, nil, nil, nil, logger),
		vehiclebiz.NewManufacturerUseCase(vrepo, logger),
		vehiclebiz.NewFirmwareUseCase(vrepo, logger),
		vehiclebiz.NewDevicePerformanceUseCase(vrepo, logger),
		vehiclebiz.NewDeviceFaultUseCase(vrepo, logger),
		vehiclebiz.NewDeviceStatsUseCase(vrepo, logger),
		vehiclebiz.NewVehicleQueryUseCase(vrepo, logger),
		vehiclebiz.NewCommandUseCase(vrepo, nil, logger),
		vehiclebiz.NewRecordQueryUseCase(vrepo),
		vehiclebiz.NewBlacklistUseCase(vrepo, logger),
		logger)

	// 租户中间件：模拟网关注入租户上下文
	tenantMW := func(h middleware.Handler) middleware.Handler {
		return func(c context.Context, req interface{}) (interface{}, error) {
			return h(multitenancy.ContextWithTenant(c, env.tenant), req)
		}
	}

	vlis := mustListener(t)
	vhlis := mustListener(t)
	vgs := kratosgrpc.NewServer(kratosgrpc.Listener(vlis), kratosgrpc.Middleware(tenantMW))
	vhs := kratoshttp.NewServer(kratoshttp.Listener(vhlis), kratoshttp.Middleware(tenantMW))
	vehiclev1.RegisterVehicleServiceServer(vgs, vehicleSvc)
	vehiclev1.RegisterVehicleServiceHTTPServer(vhs, vehicleSvc)
	startServer(t, "vehicle-grpc", vgs.Start)
	startServer(t, "vehicle-http", vhs.Start)
	waitReady(t, vhlis)
	env.vehicleHTTP = "http://" + vhlis.Addr().String()

	// ---------- payment ----------
	pclient, err := paymentent.Open("postgres", testDSN())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pclient.Close() })
	require.NoError(t, pclient.Schema.Create(ctx))

	pdata, pcleanup, err := paymentdata.NewData(pclient, logger)
	require.NoError(t, err)
	t.Cleanup(pcleanup)
	orderRepo := paymentdata.NewOrderRepo(pdata)
	env.orderRepo = orderRepo

	vconn, err := kratosgrpc.DialInsecure(ctx, kratosgrpc.WithEndpoint(vlis.Addr().String()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = vconn.Close() })
	vehicleClient := vehiclev1.NewVehicleServiceClient(vconn)

	paymentUC := paymentbiz.NewPaymentUseCase(
		orderRepo,
		paymentbiz.NewVehicleRecordRepoAdapter(vehicleClient),
		paymentbiz.NewGateControlAdapter(vehicleClient),
		&paymentbiz.PaymentConfig{}, nil, nil, logger)
	reconUC := paymentbiz.NewReconciliationUseCase(
		orderRepo, paymentdata.NewReconciliationRepo(pdata), nil, nil, paymentUC, logger)
	paymentSvc := paymentservice.NewPaymentService(paymentUC, reconUC, logger)

	phlis := mustListener(t)
	phs := kratoshttp.NewServer(kratoshttp.Listener(phlis), kratoshttp.Middleware(pkgmiddleware.CacheRawBody()))
	paymentv1.RegisterPaymentServiceHTTPServer(phs, paymentSvc)
	startServer(t, "payment-http", phs.Start)
	waitReady(t, phlis)
	env.paymentHTTP = "http://" + phlis.Addr().String()

	return env
}

func postJSON(t *testing.T, url, body string) map[string]interface{} {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewReader([]byte(body)))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "url: %s", url)
	var out map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&out))
	return out
}

func dataOf(t *testing.T, resp map[string]interface{}) map[string]interface{} {
	t.Helper()
	require.EqualValues(t, 0, resp["code"], "response: %v", resp)
	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "missing data in response: %v", resp)
	return data
}

// protoJSONInt64 解码 protojson 数值：kratos 对 proto.Message 使用 protojson，
// int64 字段按 proto3 JSON 约定序列化为字符串。
func protoJSONInt64(v interface{}) int64 {
	switch x := v.(type) {
	case float64:
		return int64(x)
	case int64:
		return x
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	default:
		return 0
	}
}

// TestE2EEntryToPaidExit 走通 入场 → 出场计费 → 建单(权威金额) → 结算回写 → 无感放行。
func TestE2EEntryToPaidExit(t *testing.T) {
	env := setupE2E(t)
	plate := randomPlate("京E")

	// 1. 相机上报入场
	entryResp := dataOf(t, postJSON(t, env.vehicleHTTP+"/api/v1/device/entry", fmt.Sprintf(
		`{"deviceId":"CAM001","plateNumber":%q,"plateImageUrl":"https://cam.smart-park.internal/snapshots/entry.jpg","confidence":0.95}`, plate)))
	require.Equal(t, true, entryResp["allowed"])
	require.Equal(t, true, entryResp["gateOpen"])
	recordID, ok := entryResp["recordId"].(string)
	require.True(t, ok && recordID != "")

	// 2. 未支付出场：计费(默认费率, 不足1小时5元) + 关闸提示缴费
	unpaidExit := dataOf(t, postJSON(t, env.vehicleHTTP+"/api/v1/device/exit", fmt.Sprintf(
		`{"deviceId":"CAM003","plateNumber":%q,"plateImageUrl":"https://cam.smart-park.internal/snapshots/exit.jpg","confidence":0.95}`, plate)))
	require.Equal(t, false, unpaidExit["allowed"])
	require.Equal(t, false, unpaidExit["gateOpen"])
	require.Equal(t, int64(500), protoJSONInt64(unpaidExit["finalAmount"]), "不足1小时应按默认费率收5元")

	// 3. 客户端篡改金额建单：服务端权威金额必须生效
	_ = postJSON(t, env.paymentHTTP+"/api/v1/pay/create", fmt.Sprintf(
		`{"recordId":%q,"amount":1,"payMethod":"wechat"}`, recordID))
	rid, err := uuid.Parse(recordID)
	require.NoError(t, err)
	order, err := env.orderRepo.GetOrderByRecordID(context.Background(), rid)
	require.NoError(t, err)
	require.NotNil(t, order, "订单应在支付URL生成前就已创建")
	require.Equal(t, int64(500), order.FinalAmount, "订单金额必须是服务端计费结果而非客户端自报金额")

	// 4. 模拟支付结算后的回写（payment 回调 → vehicle UpdateRecordStatus）
	statusResp := postJSON(t, env.vehicleHTTP+"/api/v1/vehicle/records/"+recordID+"/status", `{"status":"paid"}`)
	require.EqualValues(t, 0, statusResp["code"], "回写响应: %v", statusResp)

	// 5. 再次出场：已支付直接放行（无感出场）
	paidExit := dataOf(t, postJSON(t, env.vehicleHTTP+"/api/v1/device/exit", fmt.Sprintf(
		`{"deviceId":"CAM003","plateNumber":%q,"plateImageUrl":"https://cam.smart-park.internal/snapshots/exit-retry.jpg","confidence":0.95}`, plate)))
	require.Equal(t, true, paidExit["allowed"])
	require.Equal(t, true, paidExit["gateOpen"])
	require.Equal(t, int64(0), protoJSONInt64(paidExit["finalAmount"]))
}

// TestE2EExceptionPaths 覆盖正常链路之外的异常分支：
// 低置信度拒收、未知设备、无入场记录出场、黑名单拦截、
// 非法状态回写、重复建单幂等。
func TestE2EExceptionPaths(t *testing.T) {
	env := setupE2E(t)

	// enterVehicle 模拟车道相机上报入场（默认识别置信度 0.95）。
	enterVehicleWithConfidence := func(plate, device string, confidence float64) map[string]interface{} {
		return dataOf(t, postJSON(t, env.vehicleHTTP+"/api/v1/device/entry", fmt.Sprintf(
			`{"deviceId":%q,"plateNumber":%q,"plateImageUrl":"https://cam.smart-park.internal/snapshots/entry.jpg","confidence":%v}`, device, plate, confidence)))
	}
	enterVehicle := func(plate, device string) map[string]interface{} {
		return enterVehicleWithConfidence(plate, device, 0.95)
	}
	exitVehicle := func(plate, device string) map[string]interface{} {
		return dataOf(t, postJSON(t, env.vehicleHTTP+"/api/v1/device/exit", fmt.Sprintf(
			`{"deviceId":%q,"plateNumber":%q,"plateImageUrl":"https://cam.smart-park.internal/snapshots/exit.jpg","confidence":0.95}`, device, plate)))
	}

	t.Run("低置信度车牌拒收", func(t *testing.T) {
		resp := enterVehicleWithConfidence(randomPlate("京A"), "CAM001", 0.3)
		require.Equal(t, false, resp["allowed"])
		require.Equal(t, false, resp["gateOpen"])
	})

	t.Run("未知设备拒收", func(t *testing.T) {
		resp := enterVehicle(randomPlate("京A"), "CAM999")
		require.Equal(t, false, resp["allowed"])
		require.Equal(t, false, resp["gateOpen"])
	})

	t.Run("黑名单车辆拦截", func(t *testing.T) {
		plate := randomPlate("京B")
		require.NoError(t, env.vehicleRepo.CreateBlacklistEntry(
			multitenancy.ContextWithTenant(context.Background(), env.tenant),
			&vehiclebiz.BlacklistEntry{
				ID: uuid.New(), PlateNumber: plate, Reason: "盗抢车辆管控（e2e 验证场景）", Active: true,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}))
		resp := enterVehicle(plate, "CAM001")
		require.Equal(t, false, resp["allowed"])
		require.Equal(t, false, resp["gateOpen"])
	})

	t.Run("无入场记录出场不放行", func(t *testing.T) {
		resp := exitVehicle(randomPlate("京C"), "CAM003")
		require.Equal(t, false, resp["allowed"])
		require.Equal(t, false, resp["gateOpen"])
		require.NotEmpty(t, resp["displayMessage"])
	})

	t.Run("非法状态回写被拒绝", func(t *testing.T) {
		plate := randomPlate("京D")
		entryResp := enterVehicle(plate, "CAM001")
		recordID := entryResp["recordId"].(string)

		resp := postJSON(t, env.vehicleHTTP+"/api/v1/vehicle/records/"+recordID+"/status", `{"status":"hacked"}`)
		require.EqualValues(t, 500, resp["code"])

		// 不存在的记录同样报错
		resp = postJSON(t, env.vehicleHTTP+"/api/v1/vehicle/records/"+uuid.New().String()+"/status", `{"status":"paid"}`)
		require.EqualValues(t, 500, resp["code"])
	})

	t.Run("未支付重复建单只保留一笔订单", func(t *testing.T) {
		plate := randomPlate("京F")
		entryResp := enterVehicle(plate, "CAM001")
		recordID := entryResp["recordId"].(string)
		exitVehicle(plate, "CAM003") // 产生计费金额

		rid, err := uuid.Parse(recordID)
		require.NoError(t, err)

		// 第一次建单（payURL 生成失败不影响订单落库）
		_ = postJSON(t, env.paymentHTTP+"/api/v1/pay/create", fmt.Sprintf(
			`{"recordId":%q,"amount":500,"payMethod":"wechat"}`, recordID))
		// 用户没付钱，重新扫码再建单
		_ = postJSON(t, env.paymentHTTP+"/api/v1/pay/create", fmt.Sprintf(
			`{"recordId":%q,"amount":500,"payMethod":"wechat"}`, recordID))

		order, err := env.orderRepo.GetOrderByRecordID(context.Background(), rid)
		require.NoError(t, err, "重复建单不能让订单查询进入多行歧义状态")
		require.NotNil(t, order)
		require.Equal(t, "pending", order.Status)
	})

	t.Run("已支付订单重复建单幂等返回", func(t *testing.T) {
		plate := randomPlate("京G")
		entryResp := enterVehicle(plate, "CAM001")
		recordID := entryResp["recordId"].(string)
		exitVehicle(plate, "CAM003")

		// 首次建单（pending）
		_ = postJSON(t, env.paymentHTTP+"/api/v1/pay/create", fmt.Sprintf(
			`{"recordId":%q,"amount":500,"payMethod":"wechat"}`, recordID))
		order := recordIDToOrderID(t, env, recordID)

		// 模拟回调结算成功（微信支付交易号为 28 位数字字符串）
		oid, err := uuid.Parse(order)
		require.NoError(t, err)
		wechatTransactionID := fmt.Sprintf("420000%022d", time.Now().UnixNano())
		applied, err := env.orderRepo.MarkOrderPaid(context.Background(), oid, "wechat", wechatTransactionID, 500, time.Now())
		require.NoError(t, err)
		require.True(t, applied)

		// 已付订单再建单：幂等返回已有订单信息
		resp := postJSON(t, env.paymentHTTP+"/api/v1/pay/create", fmt.Sprintf(
			`{"recordId":%q,"amount":500,"payMethod":"wechat"}`, recordID))
		require.EqualValues(t, 0, resp["code"], "响应: %v", resp)
		data, ok := resp["data"].(map[string]interface{})
		require.True(t, ok)
		require.Equal(t, order, data["orderId"])
	})
}

// recordIDToOrderID 通过订单仓储把 recordId 换成订单 ID。
func recordIDToOrderID(t *testing.T, env *e2eEnv, recordID string) string {
	t.Helper()
	rid, err := uuid.Parse(recordID)
	require.NoError(t, err)
	order, err := env.orderRepo.GetOrderByRecordID(context.Background(), rid)
	require.NoError(t, err)
	require.NotNil(t, order)
	return order.ID.String()
}

// TestE2EExitWhenBillingDown 回归资损保护：billing 不可达时出场必须关闸
// 不放行（拒绝白放车，引导人工收费），入场不受影响。
func TestE2EExitWhenBillingDown(t *testing.T) {
	env := setupE2E(t, func(o *e2eOpts) { o.billingDown = true })
	plate := randomPlate("京H")

	// 入场不依赖 billing，照常放行
	entryResp := dataOf(t, postJSON(t, env.vehicleHTTP+"/api/v1/device/entry", fmt.Sprintf(
		`{"deviceId":"CAM001","plateNumber":%q,"plateImageUrl":"https://cam.smart-park.internal/snapshots/entry.jpg","confidence":0.95}`, plate)))
	require.Equal(t, true, entryResp["allowed"])

	// 出场时计费不可达：不放行、不开闸、提示人工
	unpaidExit := dataOf(t, postJSON(t, env.vehicleHTTP+"/api/v1/device/exit", fmt.Sprintf(
		`{"deviceId":"CAM003","plateNumber":%q,"plateImageUrl":"https://cam.smart-park.internal/snapshots/exit.jpg","confidence":0.95}`, plate)))
	require.Equal(t, false, unpaidExit["allowed"], "billing 不可达时禁止放行")
	require.Equal(t, false, unpaidExit["gateOpen"], "billing 不可达时禁止开闸")
	require.NotEmpty(t, unpaidExit["displayMessage"])
}
