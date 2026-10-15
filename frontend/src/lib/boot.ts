import { useEffect, useRef, useState } from "react";
import { isMyGo, runtime } from "mygo-runtime";
import { Library, Soap, events, type Boot, type Prefs } from "../mygo";

export const isMac = (() => {
  try {
    return isMyGo() ? runtime().platform === "darwin" : /Mac/.test(navigator.platform);
  } catch {
    return false;
  }
})();

if (isMac) document.documentElement.classList.add("mac");

/** Formats a MyGo accelerator ("CmdOrCtrl+Shift+V") for display. */
export function keys(acc: string): string {
  const parts = acc.split("+");
  if (isMac) {
    const map: Record<string, string> = { CmdOrCtrl: "⌘", Cmd: "⌘", Command: "⌘", Ctrl: "⌃", Control: "⌃", Alt: "⌥", Option: "⌥", Shift: "⇧" };
    const order = ["⌃", "⌥", "⇧", "⌘"];
    const mods = parts.slice(0, -1).map((p) => map[p] ?? p).sort((a, b) => order.indexOf(a) - order.indexOf(b));
    return mods.join("") + parts[parts.length - 1].toUpperCase();
  }
  return parts.map((p) => (p === "CmdOrCtrl" ? "Ctrl" : p)).join("+");
}

/** Loads the shared library data and keeps it current across windows. */
export function useBoot() {
  const [boot, setBoot] = useState<Boot | null>(null);
  const alive = useRef(true);
  useEffect(() => {
    alive.current = true;
    const load = () =>
      Soap.boot()
        .then((b) => alive.current && setBoot(b))
        .catch((e) => console.error(e));
    load();
    const off1 = events.libraryChanged.on(load);
    const off2 = events.prefsChanged.on((p) => setBoot((b) => (b ? { ...b, prefs: p } : b)));
    return () => {
      alive.current = false;
      off1();
      off2();
    };
  }, []);
  return boot;
}

/** Saves a preference change (every window hears prefsChanged). */
export function setPref<K extends keyof Prefs>(prefs: Prefs, key: K, value: Prefs[K]) {
  return Library.setPrefs({ ...prefs, [key]: value });
}
