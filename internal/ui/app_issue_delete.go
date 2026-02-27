package ui

import (
	"fmt"

	"fyne.io/fyne/v2"

	"jira-ai-generator/internal/domain"
	"jira-ai-generator/internal/logger"
	"jira-ai-generator/internal/ui/state"
)

// handleIssueDeleteRequestV2는 분석 선택기에서 발생한 이슈 삭제 요청을 처리한다.
func (a *App) handleIssueDeleteRequestV2(channelIndex int, payload map[string]interface{}, v2 *AppV2State) {
	if a == nil || v2 == nil || a.issueStore == nil {
		return
	}
	if channelIndex < 0 || channelIndex >= 3 {
		return
	}
	if payload == nil {
		return
	}

	record, ok := payload["issueRecord"].(*domain.IssueRecord)
	if !ok || record == nil {
		return
	}

	targetChannel := channelIndex
	if record.ChannelIndex >= 0 && record.ChannelIndex < 3 {
		targetChannel = record.ChannelIndex
	}

	go func(issue *domain.IssueRecord, channel int) {
		err := a.issueStore.DeleteIssueByIDAndChannel(issue.ID, channel)
		if err != nil {
			logger.Debug("handleIssueDeleteRequestV2: delete failed, issueID=%d, channel=%d, err=%v", issue.ID, channel, err)
			fyne.Do(func() {
				v2.appState.AddLog(channel, state.LogError, "삭제 실패: "+issue.IssueKey, "App")
				a.channels[channel].StatusLabel.SetText(fmt.Sprintf("삭제 실패: %s", issue.IssueKey))
			})
			return
		}

		fyne.Do(func() {
			logger.Debug("handleIssueDeleteRequestV2: delete success, issueID=%d, channel=%d", issue.ID, channel)
			v2.appState.AddLog(channel, state.LogInfo, "삭제 완료: "+issue.IssueKey, "App")
			a.channels[channel].StatusLabel.SetText(fmt.Sprintf("🗑 %s 삭제 완료", issue.IssueKey))
			v2.sidebar.RemoveHistoryItem(buildHistoryID(channel, issue.ID))

			// 현재 화면에 삭제된 이슈가 표시 중이면 함께 초기화한다.
			ch := a.channels[channel]
			if ch.CurrentDoc != nil && ch.CurrentDoc.IssueKey == issue.IssueKey {
				ch.CurrentDoc = nil
				ch.CurrentMDPath = ""
				ch.CurrentAnalysisPath = ""
				ch.CurrentPlanPath = ""
				ch.CurrentScriptPath = ""
				v2.resultPanels[channel].Reset()
			}

		})
	}(record, targetChannel)
}
