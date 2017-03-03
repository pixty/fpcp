# Frame Processor Communication Protocol (FPCP)
The Frame Processor Communication Protocol FPCP is used in video survilence scenes for desribing people observed by video camera in real time.
## Introduction
The FPCP supposes to work with the components depicted on the following diagram:
```

                     +----------+                +-----------+
                     |  Frame   |<---------------|   Scene   |
     (0)============>| Processor|    Downstream  | Processor |
    video            |          |--------------->|           |
    camera           +----------+    Upstream    +-----------+
```
On the diagram the `video camera` (VC) sends the video stream to the `Frame Processor` (FP), which receives the raw video signal handles it and can send some data to the `Scene Processor` (SP) for further processing. The data flow from FP to SP is named `Upstream`. The SP can communicate with FP in opposite to the Upstream direction. The data flow from SP to FP is named `Downstream`. 

The FPCP describes format, messages and rules for the data exchange between FP and SP in both directions (Upstream and Downstream). 

## Agreements
### Timestamps
All timestamps are in milliseconds. Value <= 0 means that the value us not defined
### Date/Time
All date-time values should be compliant to RFC 3339

## Definitions
JSON is used to describe FPCP structures, but it is for description the protocol only. The concrete implementation can use internal structures which are not a JSON messages!
### Rectangle size
Rectangle Size is a pair of integers described by the following JSON:
```
{"w": 123, "h": 132}
```
### Rectangle
All rectangles are described by the following JSON:
```
{"l": 123, "t": 123, "r": 1234, "b": 1234}
```
### Image
An image is picture described by the following JSON:
```
{
    "id": "pic1234",
    "size": {"w": 123, "h": 132}
    "timestamp": 12341234
    "data": <--- byte array data
}
```
The field `data` contains the image data. It is a concrete implementation specific how to pass the array actually. 
### Person
A person is a human face catched by FP and observed by some period of time. FP can apply any techniques for predicting that a face observed on different frames in time intends to the same person. FP applies an unique identifier to the observed person and reports that it observes the person since some time with a confidence level. FP can report same real person with different IDs if FP cannot guarantee with a confidence that this is the same person (even if it is same one in reality)

Person described by the following JSON:
```
{
    "id": "p1234",
    "firstSeenAt": 1234621834612
    "lostAt" 12348888888133
    "faces": [
        {"imgId": "pic1234", "region": {"l": 123, "t": 123, "r": 1234, "b": 1234}}, ...
    ]
}
```
if the `lostAt` is not provided it means the FP still has the person in the current scene.

### Scene
Scene is a cognitive description (or semantic) what is going on in the VideoStream at a moment. Scene is described by the following JSON:
```
{
    "timestamp": 123412341234
    "persons": [{Person1 JSON}, {Person2}...]
}
```
Scene contains a list of observed persons. FP can register some movements of persons on the scene, but the current scene is considered the same if the list of persons remains the same. So a person can be caught by FP and can move around the scene, but the scene will not be changed until the list of the scene persons is changed (somebody comes or leaves)

## Flow and data explanation
### Upstream data flow
Upstream messages are messages which are sent by FP to SP
#### Request response message
The message is send as a response on a request which was sent by downstream. The message contains response meta-information and a binary block data (if there is one associated with the response). The meta informations has the following JSON format:
```
{
    "reqId": "req1234",
    "error": 0,
    "scene": { Scene JSON}
    "person": {Person JSON}
    "image": {Image JSON}
}
```
and a block or binary information associated with the response. Non-zero `error` field indicates about an error. The following codes are known:
- error=1 - the object is not found
- error=2 - the connection is closed
- error=3 - timout. 

#### Scene change message
The scene message is formed by FP and send to SP every time when one of the following happens:
- scene is changed (the list of observed persons is changed)
- the `get scene` request is received

The message is sent in the response format with 0 length of binary data and the scene field is populated:
```
{
    "requId": "scene"
    "scene": {
        "timestamp": 123412341234
        "persons": [{Person1 JSON}, {Person2}...]
    }
}
```

### Downstream data flow
The downstream data flow contains requests which are sent by SP to FP. Every request is supposed to be answered by a response associated with the request received by upstream channel:
```
{
    "reqId": "1234",
    "scene": false,
    "imgId": "pic123",
    "personId": "pers1234"
}
```

It is SP responsibility to support uniqueness of the request id.

#### Get scene
Causes the FP will send current scene immediately:
```
{
    "reqId": "1234",
    "scene": true
}
```

#### Get image
The get image request contains request to get an image. The request has the following format:
```
{
    "reqId": "req1234",
    "imgId": "img1234"
}
```
The response will contain Image description in the meta field and image in the binary data associated with the response.

#### Get person
The get person request contains request to get an information about a person. The request has the following format:
```
{
    "reqId": "req1234",
    "personId": "p1234"
}
```
The response will contain the person information in the meta field.

## HTTP implementation 
TBD.
