# npm-unwrap

> consistent installs for shrinkwrapped npm projects

The goal of npm-unwrap is to take an npm-shrinkwrap.json file, and write all of
the modules to the `node_modules` directory, matching the structure of the JSON.

For various reasons, it is quite difficult to get npm itself to do this - usually,
calling `npm install` does dependency resolution also, which slows things down
and (on npm2) re-resolves peer dependencies, resulting in an indeterministic
module tree.

Right now, it handles tarballs (either from the npm registry or a separate
registry), and git Urls (of the "git+_repoUrl_#ref" form), and attempts to run
post-install scripts correctly.

## Usage

```sh
# put npm-unwrap on your path
cd my-project # js project with an npm-shrinkwrap.json file
rm -rf node_modules
npm-unwrap
```

## Why is this written in Go?

1. I wanted to learn Go.
2. Go programs are compiled as static binaries, making distribution easy.
3. Go has some nice design features (clean access to posix apis, easy
   concurrency) that make it a good fit for this sort of project.
4. It's at least halfway funny to use a static language, with a terrible package
   modify projects written in a dynamic language with an excellent package
   manager.

## License

Copyright 2015 Andrew Lunny

Licensed under the Apache License, Version 2.0: http://www.apache.org/licenses/LICENSE-2.0
