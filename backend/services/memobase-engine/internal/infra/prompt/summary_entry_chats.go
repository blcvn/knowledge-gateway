package prompt

// SummaryEntryChatsTemplate defines the templates for LLM #1 (Entry Chat Summary).
var SummaryEntryChatsTemplate = map[string]string{
	"en": `Please summarize the following chat messages into a concise user memo string. Focus on factual information about the user, their actions, and preferences.
Messages:
%s
Summary:`,
	"zh": `请将以下聊天记录总结为简洁的用户备忘录字符串。侧重于关于用户的客观信息、他们的行为和偏好。
聊天记录：
%s
总结：`,
}
