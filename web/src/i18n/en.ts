export const en: Record<string, string> = {
  // Top bar
  "topbar.working": "agent working",
  "topbar.stop": "Stop",
  "topbar.refresh": "Refresh",
  "topbar.settings": "Settings",
  "topbar.open_chat": "Ask agent",
  "topbar.close_chat": "Close chat",

  // Plan new form
  "plan.title": "Plan a week",
  "plan.subtitle": "Pick the first day of the menu and how many dinners you want.",
  "plan.start_date": "Start date",
  "plan.menu_runs_through": "Menu runs through {end}",
  "plan.num_dinners": "Number of dinners",
  "plan.notes": "Notes (optional)",
  "plan.notes_hint": "Kids away, fika bake, anything we should know.",
  "plan.notes_placeholder": "Noah away Thu–Sun. Extra bake for work fika Friday.",
  "plan.submit": "Plan the week",
  "plan.submitting": "Planning…",
  "plan.cancel": "Cancel",

  // Week view
  "week.add_dinner": "Add dinner",
  "week.regenerate": "Suggest new menu",
  "week.print": "Print",
  "week.notes_label": "notes",
  "week.notes_placeholder": "Week notes (kids away, extra bake…)",

  "print.preview_hint": "Preview. Use the button or Cmd/Ctrl+P to print.",
  "print.print_button": "Print",
  "print.overview": "Overview",
  "print.day": "Day",
  "print.dinner": "Dinner",
  "print.pers": "Pers",
  "print.source": "Source",
  "week.no_dinners": "No dinners planned yet.",
  "week.plan_dinners": "Plan dinners",
  "week.this_week": "This week",
  "week.retrospective": "Retrospective",
  "week.set_label": "set {label}",
  "week.edit_label": "Edit {label}",
  "week.clone_next": "Plan next week from this one",
  "week.add_dinner_prompt": "Add another dinner to the week running {start} through {end}.",
  "week.regenerate_prompt":
    "Regenerate the meal plan for the week running {start} through {end}. Replace each dinner with a fresh option, keeping the same constraints.",
  "week.plan_dinners_prompt": "Plan dinners for the week running {start} through {end}.",

  // Dinner card
  "dinner.adjust": "Adjust",
  "dinner.adjust_placeholder":
    "Swap for something different, less spicy, use pork instead, add potatoes as a side…",
  "dinner.adjust_hint": "⌘/Ctrl+Enter to send",
  "dinner.adjust_send": "Send",
  "dinner.adjust_cancel": "Cancel",
  "dinner.adjust_prompt":
    "For the dinner on {date} (dinner_id {id}, '{name}'): {request}. Use update_dinner to apply the change. A small tweak should preserve the dish; a swap means a new dish entirely.",
  "dinner.recipe": "Recipe",
  "dinner.servings": "pers",
  "dinner.untitled": "(untitled)",

  // Editable placeholders
  "editable.add_label": "Add {label}",

  // Status labels
  "status.draft": "draft",
  "status.cart_built": "ready to order",
  "status.ordered": "ordered",

  // Settings modal
  "settings.title": "Household defaults",
  "settings.language": "Language",
  "settings.language_sv": "Svenska",
  "settings.language_en": "English",
  "settings.delivery_weekday": "Delivery weekday",
  "settings.delivery_weekday_hint": "Which day groceries usually arrive.",
  "settings.order_offset": "Order offset (days)",
  "settings.order_offset_hint":
    "Negative means you order before delivery. -1 = day before, 0 = same day.",
  "settings.num_dinners": "Number of dinners",
  "settings.num_dinners_hint": "Default length of a menu.",
  "settings.default_servings": "Default servings",
  "settings.default_servings_hint": "People eating per dinner.",
  "settings.notes": "Notes",
  "settings.save": "Save",
  "settings.saving": "Saving…",
  "settings.saved": "Saved.",
  "settings.loading": "Loading…",
  "settings.close": "Close",
  "settings.theme": "Theme",
  "settings.theme_system": "System",
  "settings.theme_light": "Light",
  "settings.theme_dark": "Dark",
  "topbar.theme": "Theme",

  // Weekdays
  "weekday.monday": "Monday",
  "weekday.tuesday": "Tuesday",
  "weekday.wednesday": "Wednesday",
  "weekday.thursday": "Thursday",
  "weekday.friday": "Friday",
  "weekday.saturday": "Saturday",
  "weekday.sunday": "Sunday",

  // Chat
  "chat.title": "Chat",
  "chat.placeholder": "Tweak anything…",
  "chat.send": "Send",
  "chat.thinking": "thinking…",
  "chat.cancelled": "Cancelled.",
  "chat.clear": "Clear",
  "chat.clear_confirm": "Clear this week's chat history? This can't be undone.",
  "chat.empty":
    'Ask anything, e.g. "make Tuesday vegetarian", "Noah didn\'t like the cilantro", "regenerate the week". Actions on the menu also run through chat.',

  // Calendar popup
  "calendar.today": "Today",
  "calendar.clear": "Clear",
  "calendar.prev_month": "Previous month",
  "calendar.next_month": "Next month",

  "topbar.preferences": "Preferences",

  // Sidebar
  "sidebar.history": "Weeks",
  "sidebar.current": "Current week",
  "sidebar.empty": "No weeks yet.",
  "sidebar.dinners_short": "dinners",
  "sidebar.new_week": "New week",
  "sidebar.new_week_title": "Plan a new week",
  "sidebar.duplicate": "Duplicate",
  "sidebar.delete": "Delete",
  "sidebar.delete_confirm":
    "Delete this plan, including its dinners and cart? This can't be undone.",

  // Duplicate plan dialog
  "duplicate.title": "Duplicate plan",
  "duplicate.source": "Source: {start} → {end}",
  "duplicate.start_date": "Start date",
  "duplicate.new_period": "New period: {start} → {end}",
  "duplicate.confirm": "Duplicate",
  "duplicate.cancel": "Cancel",
  "duplicate.close": "Close",
  "duplicate.submitting": "Duplicating…",

  // Lifecycle
  "lifecycle.current": "Current status",
  "lifecycle.build_cart": "Build cart",
  "lifecycle.build_cart_prompt":
    "Build the Willys cart for the week running {start} through {end}. Follow the cart-building method in the system prompt: first willys_cart_clear, then aggregate ingredients across ALL dinners (show the list), pick one product per ingredient, then one willys_cart_add_many with the whole list. Verify with willys_cart_get at the end. When you're done, summarise what you added and mark the week as cart_built.",
  "lifecycle.mark_cart_built": "Mark cart as built",
  "lifecycle.mark_ordered": "Mark as ordered",
  "lifecycle.record_retrospective": "Record retrospective",
  "lifecycle.open_willys": "Open Willys.se",
  "lifecycle.set_status": "Change status",
  "lifecycle.retrospective_prompt":
    "Record a retrospective for the week running {start} through {end}. Per-dinner verdicts are already captured on each card. Ask me about what those don't show: pacing, portion sizes, overall balance, whether the week felt too heavy or lopsided, anything specific to carry into next week's plan. Save it as a week-level retrospective and update preferences when a lesson should stick.",

  // Cart section
  "cart.title": "Shopping cart",
  "cart.code": "Code",
  "cart.product": "Product",
  "cart.qty": "Qty",
  "cart.price": "Price",
  "cart.reason": "Note",

  // Preferences modal
  "prefs.title": "Preferences",
  "prefs.subtitle": "Cooking principles, family, sourcing, rules. Free-form markdown per category.",
  "prefs.pick_one": "Pick a category on the left.",
  "prefs.new_category": "New category",
  "prefs.delete": "Delete",
  "prefs.confirm_delete": "Delete the '{category}' category? This cannot be undone.",

  // Integrations
  "integrations.title": "Integrations",
  "integrations.subtitle":
    "API keys and shopping accounts. Secrets stored in the database; leave a masked field untouched to keep the existing value.",
  "integrations.enabled": "Enabled",
  "integrations.category_llm": "LLM provider",
  "integrations.category_shopping": "Shopping provider",
  "integrations.secret_set_hint": "Stored. Type a new value to replace it.",

  // Preferences modal (new + body placeholder)
  "prefs.new_category_placeholder": "e.g. allergies",
  "prefs.body_placeholder": "# Heading\n\nMarkdown body…",

  // Print preview
  "print.loading": "Loading…",

  // Chat drawer aria-label
  "chat.aria": "Chat",

  // Per-dinner rating
  "rating.how_was_it": "How was it?",
  "rating.your_verdict": "Verdict:",
  "rating.loved": "Loved",
  "rating.liked": "Liked",
  "rating.meh": "Meh",
  "rating.disliked": "No",
  "rating.clear": "Clear",
  "rating.notes_placeholder":
    "What did you think? (too salty, kids wouldn't eat it, perfect for lunch next day…)",

  // Week-level retrospective
  "retro.hint":
    "Pacing, portion sizes, overall balance. Whatever the per-dinner verdicts don't already capture.",
  "retro.placeholder":
    "Heavy on red meat this week, mix in more fish next. The porchetta stretched to three days.",
  "retro.saving": "Saving…",
  "retro.saved": "Saved",
  "retro.error": "Could not save",
};
