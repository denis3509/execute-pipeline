package logs
import(
	"easylog"
)

var DefaultLogger = new(easylog.Logger)

func Info(a ...interface{}) {
	DefaultLogger.Info(a...)
}
func Error(a ...interface{}) {
	DefaultLogger.Error(a...)
}
func Warning(a ...interface{}) {
	DefaultLogger.Warning(a...)
}

func Debug(a ...interface{}) {
	DefaultLogger.Debug(a...)
}
