import { WindowSetSize, WindowUnmaximise } from '../../wailsjs/runtime/runtime';
import { CenterWindow } from '../../wailsjs/go/main/App';

const WORKSPACE_WINDOW = {
  width: 1440,
  height: 900,
};

export async function setWorkspaceWindow() {
  try {
    WindowUnmaximise();
    WindowSetSize(WORKSPACE_WINDOW.width, WORKSPACE_WINDOW.height);
    await CenterWindow();
  } catch {
    // Ignore runtime calls outside Wails.
  }
}
