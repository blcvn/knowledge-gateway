package domain

type HookType string
const (
    HookSessionStart     HookType = "session_start"
    HookPromptSubmit     HookType = "prompt_submit"
    HookPreToolUse       HookType = "pre_tool_use"
    HookPostToolUse      HookType = "post_tool_use"
    HookPostToolFailure  HookType = "post_tool_failure"
    HookSessionEnd       HookType = "session_end"
    HookTaskCompleted    HookType = "task_completed"
    HookPreSubagent      HookType = "pre_subagent"
    HookPostSubagent     HookType = "post_subagent"
    HookNotification     HookType = "notification"
    HookStop             HookType = "stop"
    HookCustom           HookType = "custom"
)

type ObsType string
const (
    ObsToolCall     ObsType = "tool_call"
    ObsToolSuccess  ObsType = "tool_success"
    ObsError        ObsType = "error"
    ObsConversation ObsType = "conversation"
    ObsFileWrite    ObsType = "file_write"
    ObsFileRead     ObsType = "file_read"
    ObsSearch       ObsType = "search"
    ObsExec         ObsType = "exec"
    ObsCommit       ObsType = "commit"
    ObsBuild        ObsType = "build"
    ObsTest         ObsType = "test"
    ObsInstall      ObsType = "install"
    ObsAPI          ObsType = "api_call"
    ObsMemory       ObsType = "memory"
    ObsDecision     ObsType = "decision"
)
