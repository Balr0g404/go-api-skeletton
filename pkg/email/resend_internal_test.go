package email

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResendSender_Send_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	orig := resendAPI
	resendAPI = srv.URL
	defer func() { resendAPI = orig }()

	s := NewResendSender("test-key", "from@example.com")
	err := s.Send(Message{To: "to@example.com", Subject: "Test", HTML: "<p>Hi</p>"})
	assert.NoError(t, err)
}

func TestResendSender_Send_WithTextBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	orig := resendAPI
	resendAPI = srv.URL
	defer func() { resendAPI = orig }()

	s := NewResendSender("test-key", "from@example.com")
	err := s.Send(Message{To: "to@example.com", Subject: "Test", HTML: "<p>Hi</p>", Text: "Hi"})
	assert.NoError(t, err)
}

func TestResendSender_Send_Non2xxStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	orig := resendAPI
	resendAPI = srv.URL
	defer func() { resendAPI = orig }()

	s := NewResendSender("test-key", "from@example.com")
	err := s.Send(Message{To: "to@example.com", Subject: "Test", HTML: "<p>Hi</p>"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status 400")
}

func TestResendSender_Send_NetworkError(t *testing.T) {
	orig := resendAPI
	resendAPI = "http://127.0.0.1:1" // nothing listening
	defer func() { resendAPI = orig }()

	s := NewResendSender("test-key", "from@example.com")
	err := s.Send(Message{To: "to@example.com", Subject: "Test", HTML: "<p>Hi</p>"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "resend: send request")
}
