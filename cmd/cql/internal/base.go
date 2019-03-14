package internal

import (
	"time"

	"github.com/sirupsen/logrus"
)

var (
	WaitTxConfirmationMaxDuration time.Duration

	// ConsoleLog is logging for console.
	ConsoleLog *logrus.Logger
)
