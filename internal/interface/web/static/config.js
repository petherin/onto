// Runtime configuration for the Onto SPA, loaded as a classic script before the
// app module so window.ONTO_API_BASE is set by the time app.js reads it.
//
// The base is empty by default, so `make web` (and the embedded Go server) serve
// the SPA and its API from the same origin using relative paths — no change to
// the local dev experience. When the SPA is deployed split from the API (the
// MiniStack layout: SPA on S3 behind onto.world, API on ECS behind
// api.onto.world), the provisioning step overwrites this file with the API's
// absolute base URL so cross-origin calls resolve.
window.ONTO_API_BASE = "";
