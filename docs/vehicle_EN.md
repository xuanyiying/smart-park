# Vehicle Service

## Module Overview

The Vehicle Service is one of the core services of the Smart Park system, responsible for handling the complete vehicle entry/exit process, device management, and parking record management. This service directly interacts with hardware devices and serves as the bridge between the system and the physical world.

## Core Functions

### 1. Vehicle Entry Processing (Entry)

#### Business Process

```
Vehicle Arrival → Ground Sensor Detection → License Plate Recognition → Duplicate Entry Check → Create Entry Record → Barrier Open
```

#### Key Technical Implementation

**Duplicate Entry Prevention Mechanism**

```go
// Check for existing entry records before processing entry
existingRecord, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
if existingRecord != nil {
    // Duplicate entry for same vehicle: return existing record without creating new one
    return &v1.EntryData{
        PlateNumber:    req.PlateNumber,
        Allowed:        false,
        GateOpen:       false,
        DisplayMessage: "Vehicle already in parking lot",
    }, nil
}
```

**Technical Decision Explanation**:
- Database uniqueness constraints + business layer double-check
- Distributed locks ensure no duplicate records in concurrent scenarios
- Supports entry for vehicles without license plates (plate_number can be null)

### 2. Vehicle Exit Processing (Exit)

#### Business Process

```
Vehicle Arrival → License Plate Recognition → Query Entry Record → Monthly Pass Validation → Fee Calculation → Payment/Release → Barrier Open
```

#### Key Technical Implementation

**License Plate Matching Verification (Anti-Evasion)**

```go
// Verify corresponding entry record exists when exiting
record, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
if record == nil {
    // No entry record: possible evasion or recognition error
    return &v1.ExitData{
        PlateNumber:    req.PlateNumber,
        Allowed:        false,
        GateOpen:       false,
        DisplayMessage: "No entry record found, please handle manually",
    }, nil
}
```

**Monthly Pass Validity Verification**

```go
// Validate monthly pass validity when exiting
if vehicle.VehicleType == VehicleTypeMonthly {
    if vehicle.MonthlyValidUntil != nil && vehicle.MonthlyValidUntil.After(time.Now()) {
        finalAmount = 0  // Monthly pass valid, free exit
    } else {
        // Monthly pass expired, downgrade to temporary vehicle billing
        record.Metadata["chargeAs"] = VehicleTypeTemporary
        record.Metadata["monthlyExpired"] = true
    }
}
```

### 3. Device Management

#### Supported Device Types

| Device Type | Function | Communication Protocol |
|-------------|----------|------------------------|
| License Plate Recognition Camera | Plate recognition, image capture | HTTP/SDK |
| Barrier Controller | Open/close barrier control | MQTT/Relay |
| Ground Sensor | Vehicle detection | IO Signal |
| LED Display | Information display | HTTP/Serial |

#### Device Authentication Mechanism

The current implementation does not use signature headers (no HMAC/signature verification logic in code):

- Device identity is identified by the `device_secret` stored in the database; seed data generates a 32-byte random key per device (see `internal/vehicle/data/seed.go`)
- External access goes through gateway JWT authentication (`/api/v1/device/*` requires `Authorization: Bearer <jwt>`); no separate device signature is used between services

### 4. Distributed Lock Implementation

Redis-based distributed locks ensure data consistency in concurrent scenarios:

```go
// Lock key design (prefix smart-park:vehicle, see cmd/vehicle/main.go and pkg/lock/redis_lock.go)
// Full key example: smart-park:vehicle:lock:entry:京A12345
lockKey := fmt.Sprintf("smart-park:vehicle:lock:%s", identifier)

// Acquire lock
acquired, err := uc.lockRepo.AcquireLock(ctx, lockKey, owner, ttl)

// Release lock (using Lua script for atomicity)
err := uc.lockRepo.ReleaseLock(ctx, lockKey, owner)
```

**Application Scenarios**:
- Vehicle entry: prevents concurrent entry creating multiple records for same plate
- Vehicle exit: prevents concurrent billing generating duplicate orders for same record
- Device control: prevents concurrent conflicting commands to same barrier

## Data Models

### Core Entities

```go
// ParkingRecord - Parking record
type ParkingRecord struct {
    ID                uuid.UUID
    LotID             uuid.UUID
    VehicleID         *uuid.UUID      // Associated vehicle (null for vehicles without plates)
    PlateNumber       *string         // License plate (supports vehicles without plates)
    PlateNumberSource string          // Source: camera/manual/offline
    EntryLaneID       uuid.UUID
    EntryTime         time.Time
    EntryImageURL     *string
    ExitLaneID        *uuid.UUID
    ExitTime          *time.Time
    ExitImageURL      *string
    ExitDeviceID      *string
    ParkingDuration   int             // Parking duration (seconds)
    RecordStatus      string          // entry/exiting/exited/paid/refunded
    ExitStatus        string          // unpaid/paid/refunded/waived
    Metadata          map[string]interface{}
}

// Device - Device information
type Device struct {
    ID           uuid.UUID
    DeviceID     string          // Unique device identifier
    LotID        uuid.UUID
    DeviceType   string          // camera/gate/sensor/display
    DeviceSecret string          // HMAC key (encrypted storage)
    Enabled      bool            // Whether enabled
    Status       string          // active/offline/disabled
    LastHeartbeat *time.Time
}

// Lane - Lane information
type Lane struct {
    ID          uuid.UUID
    LotID       uuid.UUID
    LaneCode    string
    Direction   string          // entry/exit
    DeviceID    *string
}
```

## Application Scenarios

### Scenario 1: Commercial Complex Parking

**Business Characteristics**:
- High traffic volume, high concurrency during peak hours
- Multiple vehicle types (temporary, monthly, employee, VIP)
- Complex billing rules (time-based, holidays, consumption discounts)

**Solution**:
- Distributed locks ensure data consistency under high concurrency
- Flexible billing rule engine integration
- Support for multiple discount stacking

### Scenario 2: Residential Community Parking

**Business Characteristics**:
- High proportion of monthly pass vehicles
- Visitor vehicle management required
- Simple billing rules
- Cost-sensitive

**Solution**:
- Automatic monthly pass validity verification
- Automatic downgrade to temporary vehicle when monthly pass expires
- Nighttime discount period support

### Scenario 3: Unattended Parking

**Business Characteristics**:
- Fully automated operation
- Network may be unstable
- Requires offline fault tolerance

**Solution**:
- Offline mode with local caching
- Automatic synchronization after network recovery
- Automatic alerts for abnormal scenarios

## Technical Challenges and Solutions

### Challenge 1: High Concurrency Entry/Exit Processing

**Problem Description**:
During morning and evening rush hours, a single lane may reach 300-500 vehicles/hour, requiring the system to handle high concurrency requests while ensuring the same vehicle does not enter or get billed multiple times.

**Solution Comparison**:

| Solution | Concurrency | Complexity | Consistency | Applicable Scenario |
|----------|-------------|------------|-------------|---------------------|
| Distributed Lock | High | Medium | Strong | General scenarios |
| Optimistic Lock | Medium | Low | Strong | Read-heavy scenarios |
| Unique Constraint + Idempotency | High | Low | Strong | Simple scenarios |
| Message Queue | Very High | High | Eventual | Ultra-high concurrency |
| Token Bucket + Local Lock | Very High | High | Strong | Extreme concurrency |

#### Solution 1: Distributed Lock (Currently Used)

Redis-based distributed lock to ensure data consistency in concurrent scenarios:

```go
// Lock key prefix smart-park:vehicle; full key example: smart-park:vehicle:lock:entry:京A12345
acquired, err := uc.lockRepo.AcquireLock(ctx, lockKey, owner, ttl)
if !acquired {
    return fmt.Errorf("duplicate request in progress")
}
defer uc.lockRepo.ReleaseLock(ctx, lockKey, owner)
```

The lock lease defaults to 10s (`entry_exit.lock_ttl`); a watchdog goroutine renews it every TTL/3 until the critical section completes (see `internal/vehicle/biz/entry_exit.go`).

**Pros**: Simple implementation, strong consistency, supports distributed deployment
**Cons**: Depends on Redis, has network overhead

#### Solution 2: Database Optimistic Lock

Use version number to control concurrency without additional dependencies:

```go
type ParkingRecord struct {
    ID          uuid.UUID
    PlateNumber string
    RecordStatus string
    Version     int  // Version number
}

// Update on exit (optimistic lock)
result := db.Model(&ParkingRecord{}).
    Where("id = ? AND record_status = ? AND version = ?", 
        recordID, "entry", currentVersion).
    Updates(map[string]interface{}{
        "record_status": "exited",
        "version":       currentVersion + 1,
    })

if result.RowsAffected == 0 {
    return fmt.Errorf("record updated by another transaction")
}
```

**Pros**: No Redis needed, no deadlocks, suitable for read-heavy scenarios
**Cons**: Requires retry on concurrent conflicts

#### Solution 3: Database Unique Constraint + Application Idempotency

Use database unique index to prevent duplicate entry:

```sql
-- Database-level duplicate entry prevention
CREATE UNIQUE INDEX idx_unique_active_plate 
ON parking_records(plate_number) 
WHERE record_status IN ('entry', 'exiting');
```

```go
func (uc *UseCase) ProcessEntry(ctx context.Context, req *EntryRequest) (*EntryResponse, error) {
    // Generate idempotency key
    idempotencyKey := generateIdempotencyKey(req.DeviceID, req.PlateNumber, time.Now())
    
    // Check if already processed
    if cached := uc.cache.Get(ctx, idempotencyKey); cached != nil {
        return cached.(*EntryResponse), nil
    }
    
    // Try to insert record (rely on database unique constraint)
    record, err := uc.repo.CreateEntry(ctx, req)
    if err != nil {
        // Unique constraint conflict means already entered
        if isUniqueViolation(err) {
            existing, _ := uc.repo.GetActiveRecordByPlate(ctx, req.PlateNumber)
            return &EntryResponse{IsDuplicate: true, RecordID: existing.ID}, nil
        }
        return nil, err
    }
    
    // Cache result
    uc.cache.Set(ctx, idempotencyKey, record, 5*time.Minute)
    return &EntryResponse{RecordID: record.ID}, nil
}
```

**Pros**: Simple implementation, database guarantees eventual consistency
**Cons**: Unique constraint conflict exception handling cost is higher

#### Solution 4: Message Queue Async Processing

Entry requests first go into queue, processed asynchronously in sequence:

```go
func (uc *UseCase) AsyncProcessEntry(ctx context.Context, req *EntryRequest) (string, error) {
    requestID := uuid.New().String()
    
    msg := &EntryMessage{
        RequestID:   requestID,
        PlateNumber: req.PlateNumber,
        DeviceID:    req.DeviceID,
    }
    
    // Partition by plate number to ensure sequential processing for same plate
    partitionKey := hash(req.PlateNumber) % numPartitions
    err := uc.kafkaProducer.Send(ctx, "entry-topic", partitionKey, msg)
    
    return requestID, err
}
```

**Pros**: Peak shaving, high throughput, naturally guarantees ordering
**Cons**: Higher latency, increased system complexity

#### Solution 5: Combined Solution (Recommended)

For Smart Park scenarios, a multi-layer defense combined solution is recommended:

```go
func (uc *UseCase) ProcessEntry(ctx context.Context, req *EntryRequest) (*EntryResponse, error) {
    // Layer 1: Rate limiting protection
    if !uc.rateLimiter.Allow() {
        return nil, fmt.Errorf("system busy")
    }
    
    // Layer 2: Bloom filter quick check
    if uc.bloomFilter.MayContain(req.PlateNumber) {
        // May already exist, query database to confirm
        existing, _ := uc.repo.GetActiveRecordByPlate(ctx, req.PlateNumber)
        if existing != nil {
            return &EntryResponse{IsDuplicate: true, RecordID: existing.ID}, nil
        }
    }
    
    // Layer 3: Database unique constraint (final defense)
    record, err := uc.repo.CreateEntry(ctx, req)
    if err != nil {
        if isUniqueViolation(err) {
            return &EntryResponse{IsDuplicate: true}, nil
        }
        return nil, err
    }
    
    // Add to bloom filter
    uc.bloomFilter.Add(req.PlateNumber)
    
    return &EntryResponse{RecordID: record.ID}, nil
}
```

**Optimization Measures**:

1. **Database Optimization**
   ```sql
   -- Composite index optimization
   CREATE INDEX idx_parking_records_plate_entry ON parking_records(plate_number, entry_time DESC);
   CREATE INDEX idx_parking_records_lot_status ON parking_records(lot_id, record_status) WHERE record_status IN ('entry', 'exiting');
   ```

2. **Caching Strategy**
   - Hot vehicle information cache (1 hour)
   - Rate rule cache (24 hours)
   - Device status cache (5 minutes)

3. **Connection Pool Optimization**
   - PostgreSQL connection pool: max_connections=100
   - Redis connection pool: max_connections=50

### Challenge 2: License Plate Recognition Accuracy

**Problem Description**:
In real-world scenarios, license plate recognition is affected by lighting, angles, and dirt, potentially dropping accuracy below 90%.

**Solution**:

1. **Multi-Engine Fusion Recognition**
   ```go
   // Parallel local and cloud recognition calls
   results, _ := errgroup.Execute(
       func() { return localRecognizer.Recognize(image) },
       func() { return cloudRecognizer.Recognize(image) },
   )
   
   // Confidence fusion strategy
   if localResult.Confidence >= 0.9 {
       return localResult  // High confidence local result
   }
   if cloudResult.Confidence >= 0.8 {
       return cloudResult  // Cloud more accurate
   }
   ```

2. **Manual Review Fallback**
   - Low confidence (<0.8) triggers manual confirmation
   - Admin panel provides quick correction entry

3. **No-Plate Vehicle Handling**
   - Supports entry for vehicles without plates (recorded as NULL)
   - Exit matching by entry time

### Challenge 3: Network Interruption Tolerance

**Problem Description**:
Parking lot networks may be unstable, requiring the system to continue operating during network outages.

**Solution**:

1. **Offline Mode Design**
   ```go
   // Local SQLite cache on device
   type OfflineRecord struct {
       OfflineID   string
       RecordID    uuid.UUID
       DeviceID    string
       OpenTime    time.Time
       SyncStatus  string  // pending_sync/synced/sync_failed
   }
   ```

2. **Data Synchronization Mechanism**
   - Automatic offline data sync after network recovery
   - Supports resumable transfer
   - Conflict detection and resolution

3. **Degradation Strategy**
   - Allow unpaid exit during offline mode
   - Record unpaid vehicles for collection on next entry

### Challenge 4: Concurrent Billing Consistency

**Problem Description**:
The same vehicle may trigger billing at multiple exits simultaneously, requiring prevention of duplicate billing.

**Solution**:

1. **Distributed Pessimistic Lock**
   ```go
   lockKey := fmt.Sprintf("smart-park:vehicle:lock:exit:%s", recordID)
   acquired, _ := lockRepo.AcquireLock(ctx, lockKey, owner, 30*time.Second)
   if !acquired {
       return fmt.Errorf("billing in progress")
   }
   defer lockRepo.ReleaseLock(ctx, lockKey, owner)
   ```

2. **Database Optimistic Lock**
   ```sql
   UPDATE parking_records 
   SET record_status = 'exiting', payment_lock = payment_lock + 1
   WHERE id = ? AND payment_lock = ?
   ```

3. **Idempotency Design**
   - Order creation uses record_id as unique key
   - Payment callbacks use transaction_id for deduplication

## API Documentation

### Entry API

```http
POST /api/v1/device/entry
Content-Type: application/json
Authorization: Bearer <jwt>

{
  "deviceId": "lane_001",
  "plateNumber": "京A12345",
  "plateImageUrl": "https://cdn.example.com/entry/xxx.jpg",
  "confidence": 0.95,
  "timestamp": "2026-03-20T10:00:00Z"
}
```

**Response**:
```json
{
  "code": 0,
  "data": {
    "recordId": "rec_xxx",
    "plateNumber": "京A12345",
    "allowed": true,
    "gateOpen": true,
    "displayMessage": "Welcome"
  }
}
```

### Exit API

```http
POST /api/v1/device/exit
Content-Type: application/json
Authorization: Bearer <jwt>

{
  "deviceId": "lane_002",
  "plateNumber": "京A12345",
  "plateImageUrl": "https://cdn.example.com/exit/xxx.jpg",
  "confidence": 0.93,
  "timestamp": "2026-03-20T12:00:00Z"
}
```

**Response**:
```json
{
  "code": 0,
  "data": {
    "recordId": "rec_xxx",
    "plateNumber": "京A12345",
    "parkingDuration": 7200,
    "amount": 1500,
    "discountAmount": 0,
    "finalAmount": 1500,
    "allowed": true,
    "gateOpen": false,
    "displayMessage": "Please pay"
  }
}
```

## Configuration

```yaml
# configs/vehicle.yaml (excerpt; gRPC :9001 hardcoded in cmd/vehicle/main.go)
server:
  port: 8001
  timeout: 60

database:
  driver: postgres
  primary:
    source: "host=localhost user=postgres password=postgres dbname=parking port=5432 sslmode=disable"
  replica:
    source: "host=localhost user=postgres password=postgres dbname=parking port=5432 sslmode=disable"

redis:
  addr: "localhost:6379"
  password: ""
  db: 0

mqtt:
  # broker is required: gate commands are delivered over MQTT, service refuses to start without it
  broker: ""
  port: 1883
  client_id: "vehicle-service"
  username: ""
  password: ""
  tls: false
  topics:
    command: "parking/device/{device_id}/command"
    status: "parking/device/{device_id}/status"
    heartbeat: "parking/device/{device_id}/heartbeat"

entry_exit:
  # Distributed lock lease; watchdog renews every TTL/3
  lock_ttl: "10s"
  device_online_threshold: "5m"
  min_confidence: 0.7
```

## Testing Strategy

### Unit Tests

```bash
# Run vehicle service unit tests
go test ./internal/vehicle/... -v

# Run entry scenario tests
go test ./internal/vehicle/biz -run TestEntry -v

# Run exit scenario tests
go test ./internal/vehicle/biz -run TestExit -v
```

### Integration Tests

```bash
# Start test environment
docker-compose -f deploy/docker-compose.test.yml up -d

# Run integration tests
go test ./tests/e2e -tags=integration -v
```

### Load Testing

```bash
# Use wrk for load testing
wrk -t12 -c400 -d30s -s entry_test.lua http://localhost:8000/api/v1/device/entry
```

## Monitoring Metrics

| Metric | Type | Description |
|--------|------|-------------|
| vehicle_entry_total | Counter | Total entry count |
| vehicle_exit_total | Counter | Total exit count |
| vehicle_entry_duration | Histogram | Entry processing duration |
| vehicle_exit_duration | Histogram | Exit processing duration |
| device_online_status | Gauge | Device online status |
| lock_acquire_failures | Counter | Lock acquisition failure count |

## Related Documentation

- [Billing Service Documentation](billing_EN.md)
- [Payment Service Documentation](payment_EN.md)
- [Deployment Documentation](deployment.md)
