# shared-data

Starter content shipped with the app so new families can get going quickly.

## preferences/

Anonymised template files that describe a family's shopping and cooking
conventions. Copy these into a working directory, edit to match your
household, and import them with:

```sh
willys-import --from ./my-prefs
```

The importer loads each `.md` file as one row in `cooking_principles`,
using the filename (without extension) as the `category`. The agent
reads these at planning time.

You can always edit entries later via the web UI; the files are just
a convenient way to bootstrap.
