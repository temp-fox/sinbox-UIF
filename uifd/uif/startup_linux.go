//go:build !windows
// +build !windows

package uif

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const openWrtStartupPath = "/etc/init.d/ui4freedom"

// 通过可替换的命令边界测试服务管理器，不改变 OpenWrt 的命令顺序。
var startupPathExists = func(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var runStartupCommand = func(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func isOpenWrt() bool {
	return startupPathExists("/etc/openwrt_release") &&
		startupPathExists("/etc/rc.common") &&
		startupPathExists("/etc/init.d")
}

func autoStartupProcd(enable bool) error {
	if !isOpenWrt() {
		return errors.New("Linux need to use 'systemd' or OpenWrt 'procd'")
	}

	servicePath := filepath.Join(GetWorkSpace(), "uifd", GetCurrentUIFVersion(), "uif_service")
	if !startupPathExists(servicePath) {
		return fmt.Errorf("UIF service not found: %s", servicePath)
	}

	if enable {
		if err := installOpenWrtService(servicePath); err != nil {
			return err
		}
		return runStartupCommand(openWrtStartupPath, "enable")
	}

	return runStartupCommand(openWrtStartupPath, "disable")
}

func installOpenWrtService(servicePath string) error {
	content := openWrtServiceScript(servicePath)
	dir := filepath.Dir(openWrtStartupPath)
	tmp, err := os.CreateTemp(dir, ".ui4freedom-*")
	if err != nil {
		return fmt.Errorf("create OpenWrt service file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err = tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write OpenWrt service file: %w", err)
	}
	if err = tmp.Chmod(0755); err != nil {
		tmp.Close()
		return fmt.Errorf("chmod OpenWrt service file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close OpenWrt service file: %w", err)
	}
	if err = os.Rename(tmpName, openWrtStartupPath); err != nil {
		return fmt.Errorf("install OpenWrt service file: %w", err)
	}
	return nil
}

func openWrtServiceScript(servicePath string) string {
	return fmt.Sprintf(`#!/bin/sh /etc/rc.common

START=96
STOP=10
USE_PROCD=1

APP_PATH=%s

start_service() {
	[ -x "$APP_PATH" ] || return 1
	procd_open_instance
	procd_set_param command "$APP_PATH"
	procd_set_param respawn
	procd_close_instance
}
`, shellQuote(servicePath))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
