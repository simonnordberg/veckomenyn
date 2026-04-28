// Cross-component window events. Define the name in one place so the
// dispatcher and listener don't need to import each other just to share
// a string constant.

// Fires when the user manually triggers an update check from Settings.
// UpdateBanner listens and re-fetches its status so the banner reflects
// the new state inline.
export const UPDATES_REFRESHED_EVENT = "veckomenyn:updates-refreshed";
