# How to use Protobuf at AmazingTalker

Our service template generator base on [AmazingTalker/protoc-gen-svc](https://github.com/AmazingTalker/protoc-gen-svc)

## Basic Type Support

[Here is the type map we are supporting](https://github.com/AmazingTalker/protoc-gen-svc/#type-map)

Timestamp exmpale

```proto
import "google/protobuf/timestamp.proto";

message Record {
    string id = 1;
    google.protobuf.Timestamp created_at = 4 [(gogoproto.stdtime) = true, (gogoproto.customname) = "CreatedAt", (gogoproto.jsontag) = "createdAt"];
    google.protobuf.Timestamp updated_at = 5 [(gogoproto.stdtime) = true, (gogoproto.customname) = "UpdatedAt", (gogoproto.jsontag) = "updatedAt"];
}
```

## QueryString and URL Parameters

> ⚠️ We are not supporting any type transfer from string right now.

From URL parameters

```proto
import "third_party/amazingtalker/atproto.proto";

service GoAmazing {
    rpc GetRecord(GetRecordReq) returns (GetRecordRes) {
        option (google.api.http) = {
            get: "/api/records/:id"
            body: "record"
        };
    }
}

message GetRecordReq {
    string id = 1 [(gogoproto.customname) = "ID", (gogoproto.jsontag) = "id", (atproto.frparams) = "true"];
}
```

From querystring

```proto
import "third_party/amazingtalker/atproto.proto";

message ListRecordReq {
    // keys from url queryString or url params is always type of string.
    string size = 1 [(gogoproto.customname) = "PageSize", (gogoproto.jsontag) = "size", (atproto.frquery) = "true"];
    string page = 2 [(gogoproto.customname) = "Page", (gogoproto.jsontag) = "page", (atproto.frquery) = "true"];
}
```

## Golang Field and JSON Field Naming

* Struct label base on [gogoproto/extensions/moretags](https://github.com/gogo/protobuf/blob/master/test/tags/tags.proto)
* Validator logic base on [go-playground/validator](https://github.com/go-playground/validator) 

```proto
message ListRecordReq {
    int size = 1 [(gogoproto.moretags)='validate:"required"'];
    int page = 2 [(gogoproto.moretags)='validate:"required,gte=0,lte=100"'];
}
```