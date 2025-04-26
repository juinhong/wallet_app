package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = logrus.New()

func Init(production bool, logPath string) {
	// Create log directory if needed
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		fmt.Println("Error creating log directory")
		logrus.Fatal("Failed to create log directory:", err)
	}

	// Configure file logging with rotation
	fileWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // MB
		MaxBackups: 14,  // Keep 14 days of logs
		MaxAge:     14,  // Days
		Compress:   true,
	}

	// Write to both console and file
	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	Log.SetOutput(multiWriter)

	if production {
		Log.SetFormatter(&logrus.JSONFormatter{})
		Log.SetLevel(logrus.InfoLevel)
	} else {
		Log.SetFormatter(&logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: true,
		})
		Log.SetLevel(logrus.DebugLevel)
	}
}
