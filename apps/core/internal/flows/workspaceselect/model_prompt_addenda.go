package workspaceselectflow

import "strings"

// reactNativeExpoRules is the built-in "react-native-expo" skill (opt-in via .vocode skills): RN/Expo JSX and hooks.
// Helps models avoid web DOM patterns (<button onClick>) in RN/Expo files.
const reactNativeExpoRules = `

## React Native / Expo only (stack-specific UI)
When activeFile, fullFile, or targetText indicates React Native or Expo (e.g. imports from "react-native", "expo", "expo-router"; StyleSheet.create; paths like app/(tabs)/*.tsx), follow native mobile JSX — not browser DOM. The global rules above still apply: code must compile, and importLines must cover any new symbol from "react" / "react-native" / etc. that fullFile does not already import.

- Never emit HTML intrinsic elements (lowercase tag names from the DOM): no <button>, <div>, <span>, <input>, <p>, <a>, <img>, etc. Build UI from react-native / Expo primitives already used in the file (e.g. View, Text, Image, Pressable, TouchableOpacity, ScrollView) or import them; use onPress and other RN touch props, never onClick.
- React hooks and component logic: follow normal React rules for function components — call hooks only at the top level of the component (not inside loops/conditions/nested functions). Listing useState (etc.) in importLines is not enough: you must invoke the hook in the function body before the return (e.g. const [count, setCount] = useState(0);) whenever JSX or handlers use that state. Never reference count, setCount, or other hook outputs you did not obtain from a hook call in the same component.
- Keep layout and styling idiomatic (flex, StyleSheet, existing primitives in the file).
`

// BuiltinSkillText returns extra system prompt text for a `.vocode` skill id, or "" if unknown.
func BuiltinSkillText(id string) string {
	switch strings.ToLower(strings.TrimSpace(id)) {
	case "react-native-expo", "react_native_expo", "rn-expo", "expo":
		return reactNativeExpoRules
	default:
		return ""
	}
}

// StackPromptAddenda builds suffix text for scoped edit / file-create from host params.
// Builtin skills (e.g. react-native-expo) apply only when workspaceSkillIds lists them — there is no default stack addendum.
// workspacePromptAddendum is always optional project text from .vocode.
func StackPromptAddenda(workspaceSkillIds []string, workspacePromptAddendum string) string {
	custom := strings.TrimSpace(workspacePromptAddendum)
	var b strings.Builder
	for _, id := range workspaceSkillIds {
		if t := BuiltinSkillText(id); t != "" {
			b.WriteString(t)
		}
	}
	if custom != "" {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("\n## Project (.vocode)\n")
		b.WriteString(custom)
		b.WriteString("\n")
	}
	return b.String()
}
