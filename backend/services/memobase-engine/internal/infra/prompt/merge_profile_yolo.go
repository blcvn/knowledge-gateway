package prompt

// MergeProfileYoloTemplate defines the templates for LLM #3 (Merge YOLO).
var MergeProfileYoloTemplate = map[string]string{
	"en": `Given a list of existing user profiles (indexed) and a list of new facts, merge the new facts into the profiles.
Output a JSON decision object with "add" (new profiles), "update" (modifications to existing profiles referenced by index), and "delete" (indexes to remove).
Existing Profiles:
%s
New Facts:
%s
Output Decision:`,
	"zh": `给定现有用户画像列表（带索引）和新事实列表，将新事实合并到画像中。
输出一个JSON决策对象，包含 "add"（新画像），"update"（对现有画像的修改，通过索引引用）和 "delete"（要删除的索引）。
现有画像：
%s
新事实：
%s
输出决策：`,
}
