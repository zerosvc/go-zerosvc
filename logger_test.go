package zerosvc

import "go.uber.org/zap"

type testLogger struct {
	log *zap.SugaredLogger
}

func (t testLogger) Println(v ...interface{}) {
	t.log.Infoln(v...)
}

func (t testLogger) Printf(format string, v ...interface{}) {
	t.log.Infof(format, v...)
}

func getTestLogger(logger *zap.SugaredLogger) *testLogger {
	return &testLogger{
		log: logger,
	}
}
