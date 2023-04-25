// Copyright 2018 fatedier, fatedier@gmail.com
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package json

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

func tryTypeUnpack(buffer []byte, msg Message) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%s", e)
		}
	}()
	if unpackFunc := reflect.ValueOf(msg).MethodByName("Unpack"); unpackFunc.Kind() != reflect.Func {
		return errors.New("unpack func not found")
	} else {
		params := []reflect.Value{
			reflect.ValueOf(buffer),
		}
		rets := unpackFunc.Call(params)
		if len(rets) != 1 {
			return errors.New("wrong return from unpack func")
		}
		if rets[0].IsNil() {
			return nil
		}
		if ret, ok := rets[0].Interface().(error); ok {
			return ret
		}
	}
	return errors.New("invalid pack func")
}

func tryTypePack(msg Message) (buffer []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%s", e)
		}
	}()
	if packFunc := reflect.ValueOf(msg).MethodByName("Pack"); packFunc.Kind() != reflect.Func {
		return nil, errors.New("pack func not found")
	} else {
		params := []reflect.Value{}
		rets := packFunc.Call(params)
		if len(rets) != 2 {
			return nil, errors.New("invalid return from pack func")
		}
		if buf, ok := rets[0].Interface().([]byte); ok {
			if rets[1].IsNil() {
				return buf, nil
			}
			if err, ok := rets[1].Interface().(error); ok {
				return buf, err
			}
		}
	}
	return nil, errors.New("invalid pack func")
}

func (msgCtl *MsgCtl) unpack(typeByte byte, buffer []byte, msgIn Message) (msg Message, err error) {
	if msgIn == nil {
		t, ok := msgCtl.typeMap[typeByte]
		if !ok {
			err = ErrMsgType
			return
		}

		msg = reflect.New(t).Interface().(Message)
	} else {
		msg = msgIn
	}

	if tryError := tryTypeUnpack(buffer, msg); tryError == nil {
		return
	}

	err = json.Unmarshal(buffer, &msg)
	return
}

func (msgCtl *MsgCtl) pack(msg Message) ([]byte, error) {
	if buf, tryError := tryTypePack(msg); tryError == nil {
		return buf, nil
	}

	return json.Marshal(msg)
}

func (msgCtl *MsgCtl) UnPackInto(buffer []byte, msg Message) (err error) {
	_, err = msgCtl.unpack(' ', buffer, msg)
	return
}

func (msgCtl *MsgCtl) UnPack(typeByte byte, buffer []byte) (msg Message, err error) {
	return msgCtl.unpack(typeByte, buffer, nil)
}

func (msgCtl *MsgCtl) Pack(msg Message) ([]byte, error) {
	typeByte, ok := msgCtl.typeByteMap[reflect.TypeOf(msg).Elem()]
	if !ok {
		return nil, ErrMsgType
	}

	content, err := msgCtl.pack(msg)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(nil)
	buffer.WriteByte(typeByte)
	binary.Write(buffer, binary.BigEndian, int64(len(content)))
	buffer.Write(content)
	return buffer.Bytes(), nil
}
