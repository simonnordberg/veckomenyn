export const en: Record<string, string> = {
  // Home
  "home.title": "Veckomenyn",
  "home.subtitle": "Plan a week, build the cart, order, retrospect.",
  "home.plan_new": "Plan a new week",
  "home.plan_first": "Plan your first week",
  "home.your_weeks": "Your weeks",
  "home.empty_title": "Welcome.",
  "home.empty_body":
    "This is where the family's weeks live. The agent helps you plan dinners, builds the grocery cart, and learns from each week's feedback.",
  "home.loop_step_plan": "Plan dinners for the week.",
  "home.loop_step_cart": "Let the agent build the Willys cart.",
  "home.loop_step_order": "Place the order on willys.se.",
  "home.loop_step_retro": "Record a retrospective so next week is better.",

  // Top bar
  "topbar.brand_home": "Home",
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
  "week.truncate_confirm":
    "Shortening the plan will drop {count} dinner(s) past the new end date. Continue?",
  "week.unlock_confirm":
    "This will move the week back to '{target}' so the menu becomes editable again. Cart contents and ratings stay. Continue?",
  "week.locked_hint": "Menu locked while {status}. Change status below to reopen editing.",
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
  "chat.deliberating.pondering": "pondering",
  "chat.deliberating.simmering": "simmering ideas",
  "chat.deliberating.tasting": "tasting options",
  "chat.deliberating.chopping": "chopping possibilities",
  "chat.deliberating.plating": "plating a response",
  "chat.deliberating.cookbook": "consulting the cookbook",
  "chat.deliberating.marinating": "marinating thoughts",
  "chat.deliberating.whisking": "whisking up an answer",
  "chat.deliberating.browning": "browning the butter",
  "chat.deliberating.pantry": "raiding the pantry",
  "chat.deliberating.kneading": "kneading the dough",
  "chat.deliberating.garnishing": "garnishing the plate",
  "chat.deliberating.watched_pot": "watching the pot",
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
  "duplicate.source_prefix": "Source:",
  "duplicate.start_date": "Start date",
  "duplicate.new_period_prefix": "New period:",
  "duplicate.confirm": "Duplicate",
  "duplicate.cancel": "Cancel",
  "duplicate.close": "Close",
  "duplicate.submitting": "Duplicating…",

  // Lifecycle
  "lifecycle.current": "Current status",
  "lifecycle.build_cart": "Build cart",
  "lifecycle.build_cart_prompt":
    "Build the Willys cart for the plan running {start} through {end}. Follow the cart-building method in the system prompt: start with willys_cart_get and list what's already in the cart so I can confirm; incorporate those items into the plan (don't clear unless I say 'reset'). Aggregate ingredients across ALL dinners (show the list), subtract what's already in the cart, pick one product per missing ingredient, then one willys_cart_add_many with the delta. Verify with willys_cart_get at the end. When you're done, summarise what you added and mark the plan as cart_built.",
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

  // Provider fields
  "provider.api_key": "API key",
  "provider.model": "Model",
  "provider.password": "Password",
  "provider.anthropic.name": "Anthropic",
  "provider.anthropic.model_hint":
    "Sonnet 4.6 is the balanced default. Haiku is ~5x cheaper but weaker at creative meal planning. Opus is ~3-5x more expensive but better with complex preferences.",
  "provider.anthropic.haiku": "Claude Haiku 4.5 ($1/$5, fastest)",
  "provider.anthropic.sonnet": "Claude Sonnet 4.6 ($3/$15, balanced)",
  "provider.anthropic.opus": "Claude Opus 4.7 ($15/$75, best quality)",
  "provider.openai.name": "OpenAI",
  "provider.openai.model_hint":
    "GPT-5.4 is the latest flagship. GPT-5.1 is cheaper with good quality. GPT-4.1 is fastest and cheapest.",
  "provider.openai_compat.name": "OpenAI-compatible (local/other)",
  "provider.openai_compat.base_url": "API base URL",
  "provider.openai_compat.base_url_hint":
    "Base URL for an OpenAI-compatible API (llama.cpp, Ollama, etc.).",
  "provider.openai_compat.api_key_hint":
    "Leave empty for local backends that don't require authentication.",
  "provider.openai_compat.model_hint": "Model name as the backend expects it.",
  "provider.willys.username": "Username (YYYYMMDDNNNN)",
  "provider.test": "Test connection",
  "provider.testing": "Testing...",
  "provider.test_unknown_error": "Unknown error",

  // Preferences modal (new + body placeholder)
  "prefs.new_category_placeholder": "e.g. allergies",
  "prefs.body_placeholder": "# Heading\n\nMarkdown body…",

  // Print preview
  "print.loading": "Loading…",

  // Chat drawer aria-label
  "chat.aria": "Chat",

  // Toast notifications
  "toast.dismiss": "Dismiss",
  "toast.retry": "Retry",
  "toast.save_failed": "Save failed",
  "toast.save_failed_retrying": "Save failed; retrying…",
  "toast.changes_saved": "Changes saved",
  "toast.week_deleted": "Week deleted",
  "toast.network_error": "Network error",
  "toast.unsaved_changes": "Retrospective hasn't saved yet. Leave anyway?",

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

  // LLM usage admin page
  "usage.title": "Usage & cost",
  "usage.subtitle": "Last 30 days",
  "usage.close": "Close",
  "usage.loading": "Loading…",
  "usage.empty": "No model calls recorded yet.",
  "usage.total_cost": "Total cost",
  "usage.total_calls": "Model calls",
  "usage.total_input": "Input tokens",
  "usage.total_output": "Output tokens",
  "usage.total_cache_write": "Cache writes",
  "usage.total_cache_read": "Cache reads",
  "usage.cache_hit_rate": "Cache hit rate",
  "usage.cache_hit_hint": "Share of input tokens served from cache.",
  "usage.by_model": "By model",
  "usage.by_week": "By plan",
  "usage.by_day": "By day",
  "usage.recent_conversations": "Top conversations",
  "usage.col.model": "Model",
  "usage.col.calls": "Calls",
  "usage.col.cost": "Cost",
  "usage.col.input": "Input",
  "usage.col.cache_write": "Cache write",
  "usage.col.cache_read": "Cache read",
  "usage.col.output": "Output",
  "usage.col.plan": "Plan",
  "usage.col.conversation": "Conversation",
  "usage.col.date": "Date",
  "usage.open": "Open usage report",

  "backups.title": "Backups",
  "backups.subtitle":
    "Automatic snapshots before every upgrade, plus optional nightly dumps. Files live in ./backups on the host and survive `docker compose down -v`.",
  "backups.nightly": "Nightly automatic backup",
  "backups.nightly_hint": "Takes a pg_dump every 24h. Pre-migration snapshots run regardless.",
  "backups.nightly_disabled":
    "pg_dump isn't available in this environment, so scheduled backups can't be enabled.",
  "backups.keep": "Keep last",
  "backups.keep_unit": "nightly snapshots",
  "backups.take_now": "Take backup now",
  "backups.taking": "Taking…",
  "backups.empty": "No backups yet.",
  "backups.col.taken": "Taken",
  "backups.col.reason": "Reason",
  "backups.col.size": "Size",
  "backups.reason.pre-migration": "Pre-migration",
  "backups.reason.manual": "Manual",
  "backups.reason.nightly": "Nightly",
  "backups.download": "Download",
  "backups.delete": "Delete",
  "backups.delete_confirm": "Delete this backup? This cannot be undone.",
  "backups.restore_hint":
    "To restore: stream a downloaded file into pg_restore, e.g. `podman compose exec -T db pg_restore --clean --if-exists -U veckomenyn -d veckomenyn < your.dump`.",

  "update.available": "Update available: v{version}",
  "update.notes": "Release notes",
  "update.compare": "What changed",
  "update.copy_command": "Copy upgrade command",
  "update.copied": "Copied",
  "update.dismiss": "Dismiss",
  "update.apply": "Update now",
  "update.applying": "Updating…",
  "update.waiting": "Waiting for new version…",
  "update.apply_timeout": "Update did not finish in time. Check your container's logs.",
  "update.success": "Updated from v{from} to v{to}.",
  "update.section_title": "Updates",
  "update.section_subtitle":
    "The Update now button in the banner is always available when an update is published. Toggle this on to also apply updates automatically once a day.",
  "update.auto_label": "Apply updates automatically",
  "update.auto_hint": "Hourly check; if a newer version is published, restart with the new image.",
  "update.auto_unavailable": "Auto-update needs the Watchtower trigger sidecar. See deploy docs.",
  "update.check_now": "Check for updates",
  "update.checking": "Checking…",
  "update.up_to_date": "You're on the latest version (v{version}).",
  "update.found": "Update available: v{version}.",

  "setup.welcome_title": "Welcome to Veckomenyn",
  "setup.welcome_body":
    "Veckomenyn plans your week of dinners and builds the grocery cart. Two short steps and you're in.",
  "setup.lan_warning_title": "Keep this on your home network",
  "setup.lan_warning_body":
    "There is no authentication. Anyone who can reach this URL can read your data and spend your Anthropic balance. Run on a trusted LAN or behind Tailscale / VPN.",
  "setup.next": "Continue",
  "setup.back": "Back",
  "setup.skip": "Skip",
  "setup.finish": "Open Veckomenyn",
  "setup.step": "Step {n} of {total}",
  "setup.api_key_title": "Add your Anthropic API key",
  "setup.api_key_body":
    "Veckomenyn uses Claude for menu planning and cart building. The key is encrypted at rest with a master key the app generates for you.",
  "setup.api_key_placeholder": "sk-ant-…",
  "setup.api_key_help": "Get a key at console.anthropic.com",
  "setup.api_key_save": "Save and continue",
  "setup.api_key_saving": "Saving…",
  "setup.seed_title": "Seed starter preferences",
  "setup.seed_body":
    "Optional. Loads anonymised templates for cooking style, family routines, sourcing, and shopping rules. Edit them later under Preferences. Skip if you'd rather start blank.",
  "setup.seed_button": "Seed preferences",
  "setup.seed_seeding": "Seeding…",
  "setup.seed_done": "Seeded {n} files",
};
