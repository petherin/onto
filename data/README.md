# data/

This directory holds `locations.json`, the runtime save file for your Onto
session. It is generated automatically the first time you save (either via
the explicit `save` command or on clean exit) and is **not** committed to
git — see `.gitignore`.

A fresh checkout has no `locations.json` here yet. That's expected: the app
falls back to a small, hand-authored starter universe (see
`internal/interface/cli/default_universe.go`) whenever this file is missing,
so you always get a sane starting point without any previously-generated
exploration data (and its accumulated bugs/bloat) baked in.

If you want to point the app at a different save file, set the
`ONTO_DATA_FILE` environment variable.
