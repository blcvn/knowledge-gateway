package prompt

// ExtractProfileTemplate defines the templates for LLM #2 (Extract Topics).
var ExtractProfileTemplate = map[string]string{
	"en": `Based on the following user memo and profile schema, extract structured profile facts.
Return a JSON object containing "fact_contents" (list of strings) and "fact_attributes" (list of objects with "topic" and "sub_topic" matching the contents).
User Memo:
%s
Profile Schema:
%s
Output:`,
	"zh": `基于以下用户备忘录和画像模式，提取结构化的画像事实。
返回一个JSON对象，包含 "fact_contents"（字符串列表）和 "fact_attributes"（包含匹配内容的 "topic" 和 "sub_topic" 的对象列表）。
用户备忘录：
%s
画像模式：
%s
输出：`,
}
