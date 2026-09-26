(() => {
  const messages = {
    cs: {
      appName: "Scan Collector",
      dashboardTitle: "Scan Collector",
      recordsPageTitle: (date) => `Záznamy ${date}`,
      loginPageTitle: "Přihlášení · Scan Collector",
      recordsTitle: (date) => `Záznamy z ${date}`,
      recordCount: (count) => `Celkem ${count} ${count === 1 ? "záznam" : "záznamů"}`,
      logout: "Odhlásit",
      backToSets: "← Přehled sad",
      downloadCSV: "Stáhnout CSV",
      copyAll: "Kopírovat všechen obsah",
      time: "Čas",
      content: "Obsah",
      format: "Formát",
      device: "Zařízení",
      copy: "Kopírovat",
      emptyRecords: "Tato sada zatím neobsahuje žádné záznamy.",
      setsIntro: "Přehled uložených sad podle data",
      date: "Datum",
      recordCountHeader: "Počet záznamů",
      viewRecords: "Zobrazit záznamy →",
      emptySets: "Zatím nejsou uložené žádné sady.",
      copied: "Zkopírováno",
      contentCopied: "Obsah zkopírován",
      clipboardUnavailable: "Kopírování není v tomto prohlížeči dostupné",
      loginIntro: "Přihlaste se pro zobrazení uložených skenů.",
      username: "Uživatelské jméno",
      password: "Heslo",
      extendedLogin: "Prodloužené přihlášení",
      login: "Přihlásit se",
      loginError: "Neplatné uživatelské jméno nebo heslo.",
      switchLanguage: "Přepnout jazyk na angličtinu",
    },
    en: {
      appName: "Scan Collector",
      dashboardTitle: "Scan Collector",
      recordsPageTitle: (date) => `Records ${date}`,
      loginPageTitle: "Sign in · Scan Collector",
      recordsTitle: (date) => `Records from ${date}`,
      recordCount: (count) => `${count} ${count === 1 ? "record" : "records"}`,
      logout: "Log out",
      backToSets: "← All sets",
      downloadCSV: "Download CSV",
      copyAll: "Copy all content",
      time: "Time",
      content: "Content",
      format: "Format",
      device: "Device",
      copy: "Copy",
      emptyRecords: "This set does not contain any records yet.",
      setsIntro: "Saved sets by date",
      date: "Date",
      recordCountHeader: "Records",
      viewRecords: "View records →",
      emptySets: "No saved sets yet.",
      copied: "Copied",
      contentCopied: "Content copied",
      clipboardUnavailable: "Clipboard access is not available in this browser",
      loginIntro: "Sign in to view saved scans.",
      username: "Username",
      password: "Password",
      extendedLogin: "Stay signed in longer",
      login: "Sign in",
      loginError: "Invalid username or password.",
      switchLanguage: "Switch language to Czech",
    },
  };

  let language = "en";

  function t(key, ...args) {
    const value = messages[language][key] ?? messages.en[key] ?? key;
    return typeof value === "function" ? value(...args) : value;
  }

  function setLanguage(nextLanguage, remember = true) {
    language = nextLanguage === "cs" ? "cs" : "en";
    document.documentElement.lang = language;

    document.querySelectorAll("[data-i18n]").forEach((element) => {
      const key = element.dataset.i18n;
      if (key === "recordsTitle") {
        element.textContent = t(key, element.dataset.date);
      } else if (key === "recordCount") {
        element.textContent = t(key, Number(element.dataset.count));
      } else {
        element.textContent = t(key);
      }
    });

    const pageTitle = document.querySelector("title[data-i18n-page]");
    if (pageTitle) {
      const page = pageTitle.dataset.i18nPage;
      pageTitle.textContent = page === "records"
        ? t("recordsPageTitle", pageTitle.dataset.date)
        : page === "login" ? t("loginPageTitle") : t("dashboardTitle");
    }

    document.querySelectorAll("[data-language-switch]").forEach((button) => {
      button.textContent = language === "cs" ? "🇨🇿 Čeština" : "🇬🇧 English";
      button.setAttribute("aria-label", t("switchLanguage"));
      button.title = t("switchLanguage");
      button.onclick = () => setLanguage(language === "cs" ? "en" : "cs");
    });

    if (remember) {
      try {
        localStorage.setItem("scan-collector-language", language);
      } catch (_) {}
    }
  }

  window.scanCollectorI18n = { t };

  let savedLanguage = null;
  try {
    savedLanguage = localStorage.getItem("scan-collector-language");
  } catch (_) {}
  const preferredLanguages = navigator.languages?.length
    ? navigator.languages
    : [navigator.language || "en"];
  const browserLanguage = preferredLanguages
    .map((value) => value.toLowerCase())
    .find((value) => value.startsWith("cs") || value.startsWith("en")) || "en";
  setLanguage(savedLanguage || (browserLanguage.startsWith("cs") ? "cs" : "en"), false);
})();
