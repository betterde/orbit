package checker

import "go.uber.org/zap"

type StatusHandler struct {
	inner                  CheckNotifier
	logger                 *zap.SugaredLogger
	successBeforePassing   int
	successCounter         int
	failuresBeforeWarning  int
	failuresBeforeCritical int
	failuresCounter        int
}

// NewStatusHandler set counters values to threshold in order to immediately update status after first check.
func NewStatusHandler(inner CheckNotifier, logger *zap.SugaredLogger, successBeforePassing, failuresBeforeWarning, failuresBeforeCritical int) *StatusHandler {
	return &StatusHandler{
		logger:                 logger,
		inner:                  inner,
		successBeforePassing:   successBeforePassing,
		successCounter:         successBeforePassing,
		failuresBeforeWarning:  failuresBeforeWarning,
		failuresBeforeCritical: failuresBeforeCritical,
		failuresCounter:        failuresBeforeCritical,
	}
}

func (s *StatusHandler) updateCheck(status, output string) {
	if status == HealthPassing || status == HealthWarning {
		s.successCounter++
		s.failuresCounter = 0
		if s.successCounter >= s.successBeforePassing {
			s.logger.Debug("Check status updated", "status", status)
			s.inner.UpdateCheck(status, output)
			return
		}
		s.logger.Warn("Check passed but has not reached success threshold",
			"status", status,
			"success_count", s.successCounter,
			"success_threshold", s.successBeforePassing,
		)
	} else {
		s.failuresCounter++
		s.successCounter = 0
		if s.failuresCounter >= s.failuresBeforeCritical {
			s.logger.Warn("Check is now critical", "check")
			s.inner.UpdateCheck(status, output)
			return
		}
		// Defaults to same value as failuresBeforeCritical if not set.
		if s.failuresCounter >= s.failuresBeforeWarning {
			s.logger.Warn("Check is now warning", "check")
			s.inner.UpdateCheck(HealthWarning, output)
			return
		}
		s.logger.Warn("Check failed but has not reached warning/failure threshold",
			"status", status,
			"failure_count", s.failuresCounter,
			"warning_threshold", s.failuresBeforeWarning,
			"failure_threshold", s.failuresBeforeCritical,
		)
	}
}
