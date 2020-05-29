# DoujinRenamer
This Go scipt renames files downloaded from FANZA or DLsite to "[MAKER] TITLE.zip".

You must obtain [DMM-api-id](https://affiliate.dmm.com/) and DMM-affiliate-id, and write them into `settings.json` as following.

``` js
{
    "api_id": "API_ID",
    "affiliate_id": "AFFILIATE_ID"
}
```

## How to use
```
DoujinRenamer.exe [DIR_CONTAINING_TARGET_FILES]
```

```
> DoujinRenamer.exe -h
Usage of DoujinRenamer.exe:
  -e    Execute renaming
  -r    Visit directories recursively
  -s string
        Path to settings.json (default "settings.json")
```
