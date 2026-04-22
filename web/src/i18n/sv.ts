export const sv: Record<string, string> = {
  // Top bar
  "topbar.working": "agenten arbetar",
  "topbar.stop": "Stoppa",
  "topbar.refresh": "Uppdatera",
  "topbar.settings": "Inställningar",
  "topbar.open_chat": "Öppna chatt",
  "topbar.close_chat": "Stäng chatt",

  // Plan new form
  "plan.title": "Planera en vecka",
  "plan.subtitle": "Välj menyns första dag och antal middagar.",
  "plan.start_date": "Startdatum",
  "plan.menu_runs_through": "Menyn löper till {end}",
  "plan.num_dinners": "Antal middagar",
  "plan.notes": "Noteringar (valfritt)",
  "plan.notes_hint": "Barn borta, fika, annat vi bör känna till.",
  "plan.notes_placeholder": "Noah borta tor–sön. Extra bak till fredagsfika på jobbet.",
  "plan.submit": "Planera veckan",
  "plan.submitting": "Planerar…",
  "plan.prompt": "Planera {dinners} middagar, från {start} till {end}.",

  // Week view
  "week.add_dinner": "Lägg till middag",
  "week.regenerate": "Gör om veckan",
  "week.print": "Skriv ut",
  "week.notes_label": "noteringar",
  "week.notes_placeholder": "Vecknoteringar (barn borta, extra bak…)",

  "print.preview_hint":
    "Förhandsvisning — klicka knappen eller tryck Cmd/Ctrl+P för att skriva ut.",
  "print.print_button": "Skriv ut",
  "print.overview": "Översikt",
  "print.day": "Dag",
  "print.dinner": "Middag",
  "print.pers": "Pers",
  "print.source": "Källa",
  "week.no_dinners": "Inga middagar planerade ännu.",
  "week.plan_dinners": "Planera middagar",
  "week.this_week": "Den här veckan",
  "week.retrospective": "Återblick",
  "week.set_label": "sätt {label}",
  "week.edit_label": "Redigera {label}",
  "week.add_dinner_prompt":
    "Lägg till ytterligare en middag till veckan som löper från {start} till {end}.",
  "week.regenerate_prompt":
    "Gör om menyn för veckan som löper från {start} till {end}. Ersätt varje middag med ett nytt alternativ, med samma förutsättningar.",
  "week.plan_dinners_prompt": "Planera middagar för veckan som löper från {start} till {end}.",

  // Dinner card
  "dinner.adjust": "Ändra",
  "dinner.adjust_placeholder":
    "Byt till något annat, gör mindre stark, använd fläsk istället, lägg till potatis som tillbehör…",
  "dinner.adjust_hint": "⌘/Ctrl+Enter för att skicka",
  "dinner.adjust_send": "Skicka",
  "dinner.adjust_cancel": "Avbryt",
  "dinner.adjust_prompt":
    "För middagen den {date} (dinner_id {id}, ”{name}”): {request}. Använd update_dinner för att genomföra ändringen — en mindre justering bevarar rätten, ett byte innebär en helt ny rätt.",
  "dinner.recipe": "Recept",
  "dinner.servings": "pers",
  "dinner.untitled": "(utan titel)",

  // Editable placeholders
  "editable.add_label": "Lägg till {label}",

  // Status labels
  "status.draft": "utkast",
  "status.cart_built": "kundvagn byggd",
  "status.ordered": "beställd",
  "status.archived": "arkiverad",

  // Settings modal
  "settings.title": "Hushållsinställningar",
  "settings.language": "Språk",
  "settings.language_sv": "Svenska",
  "settings.language_en": "English",
  "settings.delivery_weekday": "Leveransdag",
  "settings.delivery_weekday_hint": "Vilken dag varorna brukar levereras.",
  "settings.order_offset": "Beställningsförskjutning (dagar)",
  "settings.order_offset_hint":
    "Negativt innebär att du beställer före leverans. -1 = dagen innan, 0 = samma dag.",
  "settings.num_dinners": "Antal middagar",
  "settings.num_dinners_hint": "Standardlängd för en meny.",
  "settings.default_servings": "Standardportioner",
  "settings.default_servings_hint": "Antal personer per middag.",
  "settings.notes": "Noteringar",
  "settings.save": "Spara",
  "settings.saving": "Sparar…",
  "settings.saved": "Sparat.",
  "settings.loading": "Laddar…",
  "settings.close": "Stäng",
  "settings.theme": "Tema",
  "settings.theme_system": "System",
  "settings.theme_light": "Ljust",
  "settings.theme_dark": "Mörkt",
  "topbar.theme": "Tema",

  // Weekdays
  "weekday.monday": "Måndag",
  "weekday.tuesday": "Tisdag",
  "weekday.wednesday": "Onsdag",
  "weekday.thursday": "Torsdag",
  "weekday.friday": "Fredag",
  "weekday.saturday": "Lördag",
  "weekday.sunday": "Söndag",

  // Chat
  "chat.title": "Chatt",
  "chat.placeholder": "Justera något…",
  "chat.send": "Skicka",
  "chat.thinking": "tänker…",
  "chat.cancelled": "Avbruten.",
  "chat.clear": "Rensa",
  "chat.clear_confirm": "Rensa veckans chatthistorik? Går inte att ångra.",
  "chat.empty":
    "Fråga vad som helst — ”gör tisdag vegetarisk”, ”Noah tyckte inte om koriander”, ”gör om veckan”. Knapparna på menyn går också via chatten.",

  // Calendar popup
  "calendar.today": "Idag",
  "calendar.clear": "Rensa",
  "calendar.prev_month": "Föregående månad",
  "calendar.next_month": "Nästa månad",

  "topbar.preferences": "Preferenser",

  // Sidebar
  "sidebar.history": "Veckor",
  "sidebar.current": "Aktuell vecka",
  "sidebar.empty": "Inga veckor ännu.",
  "sidebar.dinners_short": "middagar",

  // Lifecycle
  "lifecycle.current": "Aktuell status",
  "lifecycle.build_cart": "Bygg kundvagn",
  "lifecycle.build_cart_prompt":
    "Bygg Willys-kundvagnen för veckan som löper från {start} till {end}. Följ metoden i systemprompten: börja med willys_cart_clear, aggregera sedan alla ingredienser tvärs över ALLA middagar (visa listan), välj en produkt per ingrediens, skicka sedan hela listan i ett enda willys_cart_add_many-anrop. Verifiera med willys_cart_get på slutet. När du är klar, sammanfatta vad du la till och markera veckan som cart_built.",
  "lifecycle.mark_cart_built": "Markera kundvagn klar",
  "lifecycle.mark_ordered": "Markera som beställd",
  "lifecycle.record_retrospective": "Dokumentera återblick",
  "lifecycle.archive": "Arkivera",
  "lifecycle.open_willys": "Öppna Willys.se",
  "lifecycle.set_status": "Ändra status",
  "lifecycle.retrospective_prompt":
    "Dokumentera en återblick för veckan som löpte från {start} till {end}. Fråga hur varje middag blev, vad som fungerade, vad som inte gjorde det, och något vi bör komma ihåg till nästa gång. Spara noteringarna och eventuella dish-betyg som en retrospektiv.",

  // Cart section
  "cart.title": "Inköpslista",
  "cart.code": "Kod",
  "cart.product": "Vara",
  "cart.qty": "Antal",
  "cart.price": "Pris",
  "cart.reason": "Notering",

  // Preferences modal
  "prefs.title": "Preferenser",
  "prefs.subtitle": "Matlagningsprinciper, familj, inköpsrutiner. Fri markdown per kategori.",
  "prefs.pick_one": "Välj en kategori till vänster.",
  "prefs.new_category": "Ny kategori",
  "prefs.delete": "Ta bort",
  "prefs.confirm_delete": "Ta bort kategorin ”{category}”? Detta går inte att ångra.",

  // Integrations
  "integrations.title": "Integrationer",
  "integrations.subtitle":
    "API-nycklar och butikskonton. Hemligheter lagras i databasen; lämna ett maskat fält orört för att behålla befintligt värde.",
  "integrations.enabled": "Aktiverad",
  "integrations.category_llm": "LLM-leverantör",
  "integrations.category_shopping": "Handelsleverantör",
  "integrations.secret_set_hint": "Lagrat. Skriv ett nytt värde för att ersätta det.",

  // Preferences modal (new + body placeholder)
  "prefs.new_category_placeholder": "t.ex. allergier",
  "prefs.body_placeholder": "# Rubrik\n\nMarkdown-innehåll…",

  // Print preview
  "print.loading": "Laddar…",

  // Chat drawer aria-label
  "chat.aria": "Chatt",
};
