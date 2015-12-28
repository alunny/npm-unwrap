package npm

import (
  "encoding/json"
  "errors"
  "fmt"
  "io"
)

func ParseApp(r io.Reader) (app App, error error) {
  dec := json.NewDecoder(r)
  app = App{}

  // read first token = if not '{', exit
  init, err := dec.Token()
  if err != nil {
    return app, err
  }

  switch init := init.(type) {
    case json.Delim:
      if init != '{' {
        fmt.Printf("%v\n", init)
        return app, errors.New("unwrap: shrinkwrap incorrectly formed")
      }
    default:
      fmt.Printf("%T, %v\n", init, init)
      return app, errors.New("unwrap: shrinkwrap incorrectly formed")
  }

  for {
    t, err := dec.Token()
    if err == io.EOF {
      break
    }
    if err != nil {
      return app, err
    }

    switch t := t.(type) {
      case string:
        switch t {
          case "name":
            next, _ := dec.Token()
            // check errors
            if n, ok := next.(string); ok {
              app.Name = n
            }
          case "version":
            next, _ := dec.Token()
            // check errors
            if n, ok := next.(string); ok {
              app.Version = n
            }
          case "dependencies":
            break
            //deps, _ = mkDependencies(r)
            // check errors
            //app.Dependencies = deps
        }
    }
  }

  return app, nil
}
