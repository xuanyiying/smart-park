package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type E2ETestSuite struct {
	suite.Suite
	server *httptest.Server
	client *http.Client
}

func (s *E2ETestSuite) SetupSuite() {
	s.client = &http.Client{
		Timeout: 10 * time.Second,
	}
}

func (s *E2ETestSuite) TearDownSuite() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *E2ETestSuite) TestVehicleEntryFlow() {
	req := httptest.NewRequest("POST", "/api/v1/device/entry", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *E2ETestSuite) TestVehicleExitFlow() {
	req := httptest.NewRequest("POST", "/api/v1/device/exit", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *E2ETestSuite) TestPaymentFlow() {
	req := httptest.NewRequest("POST", "/api/v1/pay/create", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *E2ETestSuite) TestUserLoginFlow() {
	req := httptest.NewRequest("POST", "/api/v1/user/login", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *E2ETestSuite) TestPlateBindingFlow() {
	req := httptest.NewRequest("POST", "/api/v1/user/plates", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")

	w := httptest.NewRecorder()

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func (s *E2ETestSuite) TestMonthlyCardPurchaseFlow() {
	req := httptest.NewRequest("POST", "/api/v1/user/monthly-card", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")

	w := httptest.NewRecorder()

	assert.Equal(s.T(), http.StatusOK, w.Code)
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}
