# go-index
A more useable go index

TODO: A lot. This is very much under development.

## API REF

### `search`

#### `search/by-path`
Search all package urls by a given search string. This endpoint is paginated.

##### Arguments:

`search/by-path?search=<STRING>&page=<INT>&limit=<INT>&suffix=<BOOL>`

| query_param | required | default | type | limitations| description |
|-------------|----------|---------|------|-------|------------|
| `search`    | **no**  | ""       | string  |          | The substring to search urls by. |
| `page`    | **no**  | 0       | int |    | The page to start on. **ZERO-INDEXED**|
| `limit`    | **no**  | 20     | int | **<=2000** | Limits the amount of urls returned per page |
| `suffix`    | **no**  | false     | bool | | Dictates if the search is substring search or suffix search (more info below) |

###### The `suffix` option.

The default behavior is to do a substring search of the url string. If you would rather implement a search by the string suffix you can enable this. Example:
```
urls = [github.com/test, gitlab.com/test, ...]

GET search/by-path/search=test
=> [github.com/test, gitlab.com/test]

GET search/by-path/search=test&suffix=true
=> []

GET search/by-path/search=git&suffix=true
=> [github.com/test, gitlab.com/test]
```

##### Response on success:
```
{
    entries: []string
    nextPage: int
}
```

##### Response on error:
```
{
    message: string
}
```

##### How pagination works:

Pages are determined using a straightforward `offset = page*limit`. This means if you wanted to get 100 results in 2 pages you would perform the following queries:

```
GET search/by-path?search=github.com&limit=50
=> Results 0-50
GET search/by-path?search=github.com&limit=50&page=1
=> Results 50-100
```
