package npm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

func populateModule(m *Module, dec *json.Decoder) (error error) {
	// read first token = if not '{', exit
	init, err := dec.Token()
	if err != nil {
		return err
	}

	switch init := init.(type) {
	case json.Delim:
		if init != '{' {
			fmt.Printf("%v\n", init)
			return errors.New("unwrap: shrinkwrap incorrectly formed")
		}
	default:
		fmt.Printf("%T, %v\n", init, init)
		return errors.New("unwrap: shrinkwrap incorrectly formed")
	}

	for {
		t, err := dec.Token()
		if err != nil {
			return err
		}

		// fmt.Printf("populateModule: %T, %v\n", t, t)

		switch t := t.(type) {
		case string:
			switch t {
			case "version":
				next, _ := dec.Token()
				// check errors
				if n, ok := next.(string); ok {
					m.Version = n
				}
			case "from":
				next, _ := dec.Token()
				// check errors
				if n, ok := next.(string); ok {
					m.From = n
				}
			case "resolved":
				next, _ := dec.Token()
				// check errors
				if n, ok := next.(string); ok {
					m.Resolved = n
				}
			case "dependencies":
				deps, _ := mkDependencies(dec)
				// check errors
				m.Dependencies = deps
			}
		case json.Delim:
			if t == '}' {
				return nil
			} else {
				return errors.New("unwrap: unexpected JSON token")
			}
		}
	}
}

func mkDependencies(dec *json.Decoder) (deps []Module, error error) {
	// read first token = if not '{', exit
	init, err := dec.Token()
	if err != nil {
		return deps, err
	}

	switch init := init.(type) {
	case json.Delim:
		if init != '{' {
			fmt.Printf("%v\n", init)
			return deps, errors.New("unwrap: shrinkwrap incorrectly formed")
		}
	default:
		fmt.Printf("%T, %v\n", init, init)
		return deps, errors.New("unwrap: shrinkwrap incorrectly formed")
	}

	for {
		t, err := dec.Token()
		if err != nil {
			return deps, err
		}

		// fmt.Printf("mkDependencies: %T, %v\n", t, t)

		switch t := t.(type) {
		case string:
			m := Module{Name: t}
			err := populateModule(&m, dec)
			// dependency name
			if err != nil {
				return deps, err
			}
			deps = append(deps, m)

		case json.Delim:
			if t == '}' {
				return deps, nil
			} else {
				return deps, errors.New("unwrap: unexpected JSON token")
			}
		}
	}
}

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
		// fmt.Printf("ParseApp: %T, %v\n", t, t)

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
				deps, _ := mkDependencies(dec)
				// check errors
				app.Dependencies = deps
			}
		case json.Delim:
			if t == '}' {
				break
			} else {
				return app, errors.New("unwrap: unexpected JSON token")
			}
		}
	}

	return app, nil
}
