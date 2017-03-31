Package templateutils provides a set of functions that are designed to
make it easier for developers to add template based scripting to their
command line tools.

When writing a command line tool that responds to users queries by outputting
data, it's nice to allow the users to provide a template script so that they
can extract exactly the data they are interested in. This can be useful both
when visually inspecting the data and also when invoking command line tools
in scripts. The best example of this is go list which allows users to pass a
template script to extract interesting information about Go packages. For
example,

```
go list -f '{{range .Imports}}{{println .}}{{end}}'
```

prints all the imports of the current package.

The aim of this package is to make it easier for developers to add template
scripting support to their tools and easier for users of these tools to
extract the information they need.   It does this by augmenting the
templating language provided by the standard library package text/template in
two ways:

1. It auto generates descriptions of the data structures passed as
input to a template script for use in help messages.  This ensures
that help usage information is always up to date with your source code.

2. It provides a suite of convenience functions to make it easy for
script writers to extract the data they need.  There are functions for
sorting, selecting rows and columns and generating nicely formatted
tables.

For example, if a program passed a slice of structs containing stock
data to a template script, we could use the following script to extract
the names of the 3 stocks with the highest trade volume.

```
{{select (head (sort . "Volume" "dsc") 3) "Name"}}
```

The functions head, sort and select are provided by this package.
