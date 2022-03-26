# Proposal: http: Add ServeFSFile and ServeFSContent

## Abstract

To better interoperate with `io/fs`, I propose adding two functions to `net/http`.

* `ServeFSFile`, the `io/fs`-based analogue to `ServeFile`.

      func ServeFSFile(w ResponseWriter, r *Request, fsys fs.FS, name string)

* `ServeFSContent`, the `io/fs`-based analogue to `ServeContent`.

      func ServeFSContent(w ResponseWriter, r *Request, info fs.FileInfo, content io.Reader)

## Background

The `net/http` package provides three built-in ways of serving files:

* `ServeFile`, which serves a file by name
* `ServeContent`, which serves a file from an `io.ReadSeeker` and some additional metadata
* `FileSystem`, which is turned into a `Handler` using `FileServer`

These were written before the `io/fs` package existed and do not work with those interfaces.

As part of adding `io/fs`, `net/http` gained `FS` which converts an `io.FS` into a `FileSystem`.

However, `ServeFile` and `ServeContent` have no `fs.FS`-based equivalents.

## Proposal

### `ServeFSFile`

`ServeFile` lets the caller easily serve the contents of a single file from the OS file system.

`ServeFSFile` lets the caller do the same for an `fs.FS`.

```
// ServeFSFile replies to the request with the contents
// of the named file or directory from the file system fsys.
//
// If the provided file ... [rest of doc is the same as ServeFile]
func ServeFSFile(w ResponseWriter, r *Request, fsys fs.FS, name string)
```

Both of these functions take a filename. The name passed to `ServeFile` is OS-specific; the name passed to `ServeFSFile` follows the `io/fs` convention (slash-separated paths).

### `ServeFSContent`

`ServeContent` is a lower-level function intended to serve the content of any file-like object. Unfortunately, it is not compatible with `io/fs`.

`ServeContent` takes an `io.ReadSeeker`; seeking is used to determine the size of the file. An `fs.File` is not (necessarily) a `Seeker`. However, the `fs.FileInfo` interface provides the file's size as well as name and modification time.

Therefore, instead of

    name string, modtime time.Time, content io.ReadSeeker

we can pass in

    info fs.FileInfo, content io.Reader

The behavior of `ServeFSContent` is otherwise the same as `ServeContent`:

```
// ServeFSContent replies to the request using the content in the
// provided Reader. The main benefit of ServeFSContent over io.Copy
// is that it handles Range requests properly, sets the MIME type, and
// handles If-Match, If-Unmodified-Since, If-None-Match, If-Modified-Since,
// and If-Range requests.
//
// ServeFSContent uses info to learn the file's name, modification time, and size.
// The size must be accurate but other attributes may have zero values.
//
// If the response's Content-Type header is not set, ServeFSContent
// first tries to deduce the type from name's file extension and,
// if that fails, falls back to reading the first block of the content
// and passing it to DetectContentType.
// The name is otherwise unused; in particular it can be empty and is
// never sent in the response.
//
// If the modification time is not the zero time or Unix epoch,
// ServeFSContent includes it in a Last-Modified header in the response.
// If the request includes an If-Modified-Since header, ServeFSContent uses
// the modification time to decide whether the content needs to be sent at all.
//
// If the caller has set w's ETag header formatted per RFC 7232, section 2.3,
// ServeFSContent uses it to handle requests using If-Match, If-None-Match,
// or If-Range.
func ServeFSContent(w ResponseWriter, r *Request, info fs.FileInfo, content io.Reader)
```

## Questions

### Should these functions instead be implemented outside the standard library?

It is not trivial to implement these functions outside of `net/http`. The proposed functions are building blocks upon which *other* functionality can be built; it is not possible to write these functions simply in terms of the existing `net/http` API.

The `ServeFile` and `ServeContent` functions do quite a lot of subtle work (path cleaning, redirects, translating OS errors to HTTP responses, content-type sniffing, and more). Implementing this proposal outside of `net/http` requires either copying a lot of its internal code or reimplementing a good amount of functionality (some of which comes with security implications).

I believe that we should add these proposed functions to `net/http` so that it supports `io/fs` just as well as it supports OS files.

### Should `ServeFSContent` have a different signature?

We could simplify the signature of `ServeFSContent` by having it take an `fs.File`:

    func ServeFSContent(w ResponseWriter, r *Request, f fs.File)

and then `ServeFSContent` would call `f.Stat` itself.

That's not entirely satisfying; it seems to be unusual to pass an `fs.File` around separately from an `fs.FS`, and `Close` is not used.

Another option would be to pass in all the fields explicitly. (This is the same as `ServeContent` except that instead of a `ReadSeeker` we pass in the size.) Since this is now not `io/fs`-specific at all, I gave it a new name:

    func ServeReader(w http.ResponseWriter, r *Request, name string, modtime time.Time, size int64, content io.Reader)
