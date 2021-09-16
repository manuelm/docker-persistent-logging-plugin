package main

import (
    "context"
    "encoding/binary"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
    "syscall"
    "time"

    "github.com/containerd/fifo"
    "github.com/docker/docker/api/types/backend"
    "github.com/docker/docker/api/types/plugins/logdriver"
    "github.com/docker/docker/daemon/logger"
    "github.com/docker/docker/daemon/logger/local"
    protoio "github.com/gogo/protobuf/io"
    "github.com/pkg/errors"
    "github.com/sirupsen/logrus"
)


type loggerContext struct {
    logger      logger.Logger
    info        logger.Info
    stream      io.ReadCloser
}

// driver does not perform the logging itself, but rather it manages an instance of
// loggerContext for each image it handled the logs for. So mostly plumbing and
// error handling.
type driver struct {
    mutex      sync.Mutex
    logs    map[string]*loggerContext  // maps file name to logger responsible for it.
}


func newDriver() *driver {
    return &driver{
        logs: make(map[string]*loggerContext),
    }
}


func (driver *driver) StartLogging(file string, info logger.Info) error {
    logrus.WithField("file", file).WithField("info", info).Debugf("StartLogging command received.")

    driver.mutex.Lock()
    if _, exists := driver.logs[file]; exists {
        driver.mutex.Unlock()
        return fmt.Errorf("logger for %q already exists", file)
    }
    driver.mutex.Unlock()

    if info.ContainerImageName == "" {
        return errors.New("image name not provided in logging info")
    }

    info.LogPath = filepath.Join("/var/log/docker", info.ContainerImageName)
    if err := os.MkdirAll(filepath.Dir(info.LogPath), 0755); err != nil {
        return errors.Wrap(err, "error setting up logger dir")
    }

    logger, err := local.New(info)
    if err != nil {
        return errors.Wrap(err, "error creating local logger")
    }

    steam, err := fifo.OpenFifo(context.Background(), file, syscall.O_RDONLY, 0700)
    if err != nil {
        return errors.Wrapf(err, "error opening logger file: %q", file)
    }


    driver.mutex.Lock()
    loggerCtx := &loggerContext{logger, info, steam}
    driver.logs[file] = loggerCtx
    driver.mutex.Unlock()

    go consumeLog(loggerCtx)
    return nil
}

func (driver *driver) StopLogging(file string) error {
    logrus.WithField("file", file).Debugf("StopLogging command received.")
    driver.mutex.Lock()
    loggerCtx, ok := driver.logs[file]
    if ok {
        loggerCtx.stream.Close()
        delete(driver.logs, file)
    }
    driver.mutex.Unlock()
    return nil
}

func consumeLog(loggerCtx *loggerContext) {
    dec := protoio.NewUint32DelimitedReader(loggerCtx.stream, binary.BigEndian, 1e6)
    defer dec.Close()
    var buf logdriver.LogEntry
    for {
        if err := dec.ReadMsg(&buf); err != nil {
            if err == io.EOF {
                logrus.WithField("id", loggerCtx.info.ContainerID).WithError(err).Debug("Closing logger stream.")
                loggerCtx.stream.Close()
                return
            }
            dec = protoio.NewUint32DelimitedReader(loggerCtx.stream, binary.BigEndian, 1e6)
            continue
        }
        var msg logger.Message
        msg.Line = buf.Line
        msg.Source = buf.Source
        if buf.PartialLogMetadata != nil {
            msg.PLogMetaData = &backend.PartialLogMetaData{}
            msg.PLogMetaData.ID = buf.PartialLogMetadata.Id
            msg.PLogMetaData.Last = buf.PartialLogMetadata.Last
            msg.PLogMetaData.Ordinal = int(buf.PartialLogMetadata.Ordinal)
        }
        msg.Timestamp = time.Unix(0, buf.TimeNano)

        if err := loggerCtx.logger.Log(&msg); err != nil {
            logrus.WithError(err).WithField("message", msg).Error("Error writing log message")
            continue
        }

        buf.Reset()
    }
}

func (d *driver) ReadLogs(info logger.Info, config logger.ReadConfig) (io.ReadCloser, error) {
    logrus.WithField("info", info).WithField("config", config).Debugf("ReadLogs command received.")
    reader, writer := io.Pipe()

    if info.ContainerImageName == "" {
        return reader, errors.New("image name not provided in logging info")
    }

    info.LogPath = filepath.Join("/var/log/docker", info.ContainerImageName)
    tempLogger, err := local.New(info)
    if err != nil {
        return reader, errors.Wrap(err, "error creating local logger")
    }

    logReader, _ := tempLogger.(logger.LogReader)

    go func() {
        watcher := logReader.ReadLogs(config)

        enc := protoio.NewUint32DelimitedWriter(writer, binary.BigEndian)
        defer enc.Close()
        defer watcher.ConsumerGone()

        var buf logdriver.LogEntry
        for {
            select {
            case msg, ok := <-watcher.Msg:
                if !ok {
                    writer.Close()
                    return
                }

                buf.Line = msg.Line
                buf.Partial = msg.PLogMetaData != nil
                buf.TimeNano = msg.Timestamp.UnixNano()
                buf.Source = msg.Source

                if err := enc.WriteMsg(&buf); err != nil {
                    writer.CloseWithError(err)
                    return
                }
            case err := <-watcher.Err:
                writer.CloseWithError(err)
                return
            }

            buf.Reset()
        }
    }()
    return reader, nil
}
