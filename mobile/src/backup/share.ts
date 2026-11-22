import { File, Paths } from "expo-file-system";
import { isAvailableAsync, shareAsync } from "expo-sharing";

/** Writes a text file to the cache and hands it to the system share sheet. Needs a device: nothing here runs in Jest. */
export interface Sharer {
  shareText(fileName: string, content: string): Promise<void>;
}

export const systemSharer: Sharer = {
  async shareText(fileName, content) {
    if (!(await isAvailableAsync())) {
      throw new Error("sharing is not available on this device");
    }
    const file = new File(Paths.cache, fileName);
    if (file.exists) {
      file.delete();
    }
    file.create();
    file.write(content);
    await shareAsync(file.uri, { mimeType: "application/json", dialogTitle: fileName, UTI: "public.json" });
  },
};
