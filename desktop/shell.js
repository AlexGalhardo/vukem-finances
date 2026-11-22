// Injected into the window after every page load by the desktop shell. It keeps
// the window on the app (external links go to the system browser) and sends
// downloads through the native save dialog, because not every webview can save
// an attachment by itself.
(() => {
	if (window.__VF_DESKTOP__) return;
	window.__VF_DESKTOP__ = true;

	const post = (path, body) => fetch(path, { method: "POST", body, credentials: "same-origin" });

	const openExternal = (href) => {
		post("/desktop/open-external", href).catch(() => {});
	};

	const nameOf = (response, url) => {
		const header = response.headers.get("Content-Disposition") || "";
		const utf8 = /filename\*=UTF-8''([^;]+)/i.exec(header);
		if (utf8) {
			try {
				return decodeURIComponent(utf8[1]);
			} catch {}
		}
		const plain = /filename="?([^";]+)"?/i.exec(header);
		return plain ? plain[1] : url.pathname.split("/").pop() || "download";
	};

	const save = async (url) => {
		try {
			const response = await fetch(url.href, { credentials: "same-origin" });
			if (!response.ok) throw new Error(String(response.status));
			const saved = await post("/desktop/save?name=" + encodeURIComponent(nameOf(response, url)), await response.blob());
			if (!saved.ok) throw new Error(String(saved.status));
		} catch {
			window.alert("The file could not be saved.");
		}
	};

	document.addEventListener(
		"click",
		(event) => {
			if (event.defaultPrevented || event.button !== 0) return;
			const link = event.target instanceof Element ? event.target.closest("a[href]") : null;
			if (!link) return;
			let url;
			try {
				url = new URL(link.href, location.href);
			} catch {
				return;
			}
			if (url.origin !== location.origin) {
				event.preventDefault();
				openExternal(url.href);
			} else if (link.hasAttribute("download")) {
				event.preventDefault();
				save(url);
			} else if (link.target === "_blank") {
				event.preventDefault();
				location.assign(url.href);
			}
		},
		true,
	);

	window.open = (href) => {
		try {
			const url = new URL(String(href), location.href);
			if (url.origin === location.origin) location.assign(url.href);
			else openExternal(url.href);
		} catch {}
		return null;
	};
})();
