import type { ReactNode } from "react";
import { Pressable, Text, View } from "react-native";
import { TOUCH_MIN, useUi } from "./ui";

/**
 * A list line: title, optional subtitle and a trailing slot (an amount). With
 * `onPress` it is a button, read as "title, subtitle". No fixed heights, so
 * the text can grow with the system font size.
 */
export function ListRow(props: {
  title: string;
  subtitle?: string;
  trailing?: ReactNode;
  onPress?: () => void;
  accessibilityHint?: string;
}): ReactNode {
  const ui = useUi();
  const { title, subtitle, trailing, onPress, accessibilityHint } = props;

  const body = (
    <View
      style={{
        flexDirection: "row",
        alignItems: "center",
        minHeight: TOUCH_MIN,
        paddingVertical: ui.spacing[2],
        paddingHorizontal: ui.spacing[4],
        gap: ui.spacing[3],
      }}
    >
      <View style={{ flex: 1, flexShrink: 1 }}>
        <Text style={{ color: ui.colors.text, fontSize: ui.fontSize.base }}>{title}</Text>
        {subtitle === undefined ? null : (
          <Text style={{ color: ui.colors.textMuted, fontSize: ui.fontSize.sm }}>{subtitle}</Text>
        )}
      </View>
      {trailing}
    </View>
  );

  if (onPress === undefined) {
    return body;
  }
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={subtitle === undefined ? title : `${title}, ${subtitle}`}
      accessibilityHint={accessibilityHint}
      onPress={onPress}
    >
      {body}
    </Pressable>
  );
}
