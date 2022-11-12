## Url Shortner

A lightweight Url shortner based on go-gin and mongo-db

### Features

* Generate Short Code for urls
* API Key based auth
* Supports stats like category and total clicks

### Routes

* GET ```/generateUser``` 

* POST ```/getAPIKey```

```
{
  "UserId": "foobar"
}
```

* POST ```/shorten```

```
{
  "LongUrl": "url",
  "UrlCategory": "foo",
  "UserId": "foobar",
  "Api_Key": "foo"
}
```

* POST ```/custom```

```
{
  "LongUrl": "url",
  "UrlCategory": "foo",
  "CustomCode" "johndoe"
  "UserId": "foobar",
  "Api_Key": "foo"
}
```

* GET ```/stats/:code```

* GET ```/:code```
