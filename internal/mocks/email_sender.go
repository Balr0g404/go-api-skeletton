package mocks

import (
	"github.com/Balr0g404/go-api-skeletton/pkg/email"
	"github.com/stretchr/testify/mock"
)

type EmailSender struct {
	mock.Mock
}

func (m *EmailSender) Send(msg email.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}
