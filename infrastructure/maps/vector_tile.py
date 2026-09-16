"""Minimal read-only MVT inspection for smoke checks (protobuf wire format)."""
import gzip

def fields(data):
    offset=0
    def integer():
        nonlocal offset
        value=0
        for shift in range(0,70,7):
            if offset>=len(data):raise ValueError("truncated protobuf")
            byte=data[offset];offset+=1;value|=(byte&127)<<shift
            if byte<128:return value
        raise ValueError("invalid protobuf varint")
    while offset<len(data):
        tag=integer();number,wire=tag>>3,tag&7
        if wire==0:value=integer()
        elif wire==2:
            length=integer();value=data[offset:offset+length];offset+=length
            if len(value)!=length:raise ValueError("truncated protobuf field")
        elif wire in (1,5):
            length=8 if wire==1 else 4;value=data[offset:offset+length];offset+=length
            if len(value)!=length:raise ValueError("truncated protobuf scalar")
        else:raise ValueError("unsupported protobuf wire type")
        yield number,value

def layer_counts(data):
    if data.startswith(b"\x1f\x8b"):data=gzip.decompress(data)
    counts={}
    for number,layer in fields(data):
        if number!=3:continue
        name=None;count=0
        for field,value in fields(layer):
            if field==1:name=value.decode("utf-8")
            elif field==2:count+=1
        if name:counts[name]=count
    return counts
