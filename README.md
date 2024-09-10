# go-index
A more useable go index

TODO: A lot. This is very much under development.

## API REF

### `search`

#### `search/by-path`
An inclusive(substring) search of all packges based on their URL. This endpoint is paginated, and meant to service real time searches.

##### Arguments:

`search/by-path?search=<STRING>&page=<INT>&limit=<INT>`

| query_param | required | default | type | limitations| description |
|-------------|----------|---------|------|-------|------------|
| `search`    | **no**  | ""       | string  |          | The substring to search urls by. |
| `page`    | **no**  | 0       | int |    | The page to start on. **ZERO-INDEXED**|
| `limit`    | **no**  | 20     | int | **<=2000** | Limits the amount of urls returned per page |

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
