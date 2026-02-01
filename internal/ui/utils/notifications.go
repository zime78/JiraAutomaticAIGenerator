package utils

import (
	"os/exec"
	"runtime"
)

// NotificationType 알림 유형
type NotificationType int

const (
	NotificationInfo NotificationType = iota
	NotificationSuccess
	NotificationWarning
	NotificationError
)

// Notification 시스템 알림
type Notification struct {
	Title   string
	Message string
	Type    NotificationType
}

// NotificationManager 알림 관리자
type NotificationManager struct {
	enabled bool
}

// NewNotificationManager 새 알림 관리자 생성
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		enabled: true,
	}
}

// SetEnabled 알림 활성화/비활성화
func (nm *NotificationManager) SetEnabled(enabled bool) {
	nm.enabled = enabled
}

// IsEnabled 알림 활성화 상태 확인
func (nm *NotificationManager) IsEnabled() bool {
	return nm.enabled
}

// Show 알림 표시
func (nm *NotificationManager) Show(notification Notification) error {
	if !nm.enabled {
		return nil
	}

	switch runtime.GOOS {
	case "darwin":
		return nm.showMacOS(notification)
	case "linux":
		return nm.showLinux(notification)
	case "windows":
		return nm.showWindows(notification)
	default:
		return nil
	}
}

// ShowInfo 정보 알림
func (nm *NotificationManager) ShowInfo(title, message string) error {
	return nm.Show(Notification{
		Title:   title,
		Message: message,
		Type:    NotificationInfo,
	})
}

// ShowSuccess 성공 알림
func (nm *NotificationManager) ShowSuccess(title, message string) error {
	return nm.Show(Notification{
		Title:   title,
		Message: message,
		Type:    NotificationSuccess,
	})
}

// ShowWarning 경고 알림
func (nm *NotificationManager) ShowWarning(title, message string) error {
	return nm.Show(Notification{
		Title:   title,
		Message: message,
		Type:    NotificationWarning,
	})
}

// ShowError 에러 알림
func (nm *NotificationManager) ShowError(title, message string) error {
	return nm.Show(Notification{
		Title:   title,
		Message: message,
		Type:    NotificationError,
	})
}

// showMacOS macOS 알림
func (nm *NotificationManager) showMacOS(n Notification) error {
	script := `display notification "` + n.Message + `" with title "` + n.Title + `"`
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// showLinux Linux 알림
func (nm *NotificationManager) showLinux(n Notification) error {
	urgency := "normal"
	switch n.Type {
	case NotificationWarning:
		urgency = "normal"
	case NotificationError:
		urgency = "critical"
	}

	cmd := exec.Command("notify-send", "-u", urgency, n.Title, n.Message)
	return cmd.Run()
}

// showWindows Windows 알림
func (nm *NotificationManager) showWindows(n Notification) error {
	// PowerShell을 사용한 Windows 토스트 알림
	script := `
	[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
	[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null
	$template = "<toast><visual><binding template='ToastText02'><text id='1'>` + n.Title + `</text><text id='2'>` + n.Message + `</text></binding></visual></toast>"
	$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
	$xml.LoadXml($template)
	$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
	[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("JiraAIGenerator").Show($toast)
	`
	cmd := exec.Command("powershell", "-Command", script)
	return cmd.Run()
}

// QuickNotify 간단한 알림 (제목과 메시지만)
func QuickNotify(title, message string) error {
	nm := NewNotificationManager()
	return nm.ShowInfo(title, message)
}

// NotifyProcessComplete 프로세스 완료 알림
func NotifyProcessComplete(issueKey string, success bool) error {
	nm := NewNotificationManager()
	if success {
		return nm.ShowSuccess("분석 완료", issueKey+" 이슈 분석이 완료되었습니다.")
	}
	return nm.ShowError("분석 실패", issueKey+" 이슈 분석 중 오류가 발생했습니다.")
}
