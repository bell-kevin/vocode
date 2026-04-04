package workspaceselectflow

// reactNativeExpoRules is appended to scoped-edit and file-create model system prompts.
// Without it, models often emit web React DOM patterns (<button onClick>) in RN/Expo files.
const reactNativeExpoRules = `

React Native / Expo: When activeFile or the code you are editing (targetText or numberedSnippet) indicates React Native or Expo — for example imports from "react-native", "expo", or "expo-router"; components such as View, Text, Image, Pressable, TouchableOpacity, ScrollView, ThemedView, ThemedText, StyleSheet.create; or paths like app/(tabs)/*.tsx — you MUST follow React Native rules, not browser React DOM:
- Never emit HTML intrinsic elements: no <button>, <div>, <span>, <input>, <p>, etc.
- For tappable UI use Pressable, TouchableOpacity, or Button from "react-native" (or the same primitives already used in the file). Handlers must be onPress (and other RN touch props), never onClick.
- Keep layout and styling idiomatic for React Native (flex, StyleSheet, existing themed components) unless that section of the file clearly targets web-only JSX.
- Do not use the global React import; prefer destructured imports from "react" or "react-native".
- Do not add imports unless specifically asked
`
