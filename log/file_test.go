package log

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

const (
	filePrefix = "test"
	fileSuffix = ".log"
	gzipSuffix = ".gz"
)

func TestFile(t *testing.T) {
	p, c := NewFilePlugin(filePrefix+fileSuffix, zapcore.DebugLevel)
	logger := NewLogger(p)
	b := make([]byte, 10000)
	count := 10000
	for count > 0 {
		count--
		logger.Info(string(b))
	}
	err := c.Close()
	require.NoError(t, err)
	// 等待lumberjack压缩日志文件
	time.Sleep(3 * time.Second)

	fs, err := os.ReadDir(".")
	require.NoError(t, err)
	var (
		gzCont,
		logCount int
	)
	for _, f := range fs {
		var name = f.Name()
		if strings.HasPrefix(name, filePrefix) {
			if strings.HasSuffix(name, fileSuffix) {
				logCount++
				assert.NoError(t, os.Remove(f.Name()))
				continue
			}
			if strings.HasSuffix(name, fileSuffix+gzipSuffix) {
				gzCont++
				assert.NoError(t, os.Remove(f.Name()))
				continue
			}
		}
	}

	require.Equal(t, 3, logCount)
	require.Equal(t, 2, gzCont)
}
