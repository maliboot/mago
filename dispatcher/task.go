package dispatcher

import (
	"bytes"
	"encoding/gob"
	eghelper "github.com/maliboot/mago/helper"
)

type Task struct {
	Name    string            `json:"name"`
	Headers map[string]string `json:"headers"`
	Payload []byte            `json:"payload"`

	options *TaskOptions
}

func NewTask(name string, headers map[string]string, payload []byte, opts ...TaskOption) *Task {
	if headers == nil {
		headers = make(map[string]string)
	} else {
		headers = eghelper.CopyMap[string, string](headers)
	}

	optsIns := TaskOptions{}
	if len(opts) != 0 {
		for _, opt := range opts {
			opt(&optsIns)
		}
	}

	if optsIns.MaxRetries == nil {
		maxTry := 2
		optsIns.MaxRetries = &maxTry
	}
	return &Task{
		Name:    name,
		Headers: headers,
		Payload: payload,
		options: &optsIns,
	}
}

func NewEmptyTask() *Task {
	return NewTask("", nil, nil)
}

func (t *Task) GetUrl() string {
	return t.Headers[taskHeadUrl]
}

func (t *Task) SetUrl(url string) *Task {
	t.Headers[taskHeadUrl] = url
	return t
}

func (t *Task) GetGuid() string {
	return t.Headers[TaskHeadGuid]
}

func (t *Task) SetGuid(guid string) *Task {
	t.Headers[TaskHeadGuid] = guid
	return t
}

func (t *Task) GetMethod() string {
	return t.Headers[TaskHeadMethod]
}

func (t *Task) SetMethod(method string) *Task {
	t.Headers[TaskHeadMethod] = method
	return t
}

func (t *Task) GetRequestId() string {
	return t.Headers[TaskHeadRequestId]
}

func (t *Task) SetRequestId(requestId string) *Task {
	t.Headers[TaskHeadRequestId] = requestId
	return t
}

func (t *Task) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(t)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GetTaskOpts(task *Task) *TaskOptions {
	return task.options
}

const (
	taskHeadUrl                = "Url"
	TaskHeadMethod             = "Method"
	TaskHeadGuid               = "Guid"
	TaskHeadServerMachineId    = "ServerMachineId"
	TaskHeadTimeout            = "Timeout"
	TaskHeadHttp2              = "Http2"
	TaskHeadHttpRedirectsCount = "HttpRedirectsCount"
	TaskHeadRequestId          = "RequestId"
)

type TaskInfo struct {
	Task
	TaskOptions

	Retries  int
	State    TaskState
	StateMsg string
}

func (ti *TaskInfo) Marshal() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(ti)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ti *TaskInfo) Unmarshal(raw []byte) (*TaskInfo, error) {
	buf := bytes.NewBuffer(raw)
	dec := gob.NewDecoder(buf)

	err := dec.Decode(ti)
	if err != nil {
		return nil, err
	}
	return ti, nil
}

func NewTaskInfo(t Task) *TaskInfo {
	return &TaskInfo{
		Task:        t,
		TaskOptions: *GetTaskOpts(&t),
		Retries:     0,
		State:       TaskStateReady,
		StateMsg:    "",
	}
}

func NewEmptyTaskInfo() *TaskInfo {
	t := NewTask("", nil, nil)
	return NewTaskInfo(*t)
}

type TaskState int

const (
	TaskStateReady      TaskState = iota // 未处理
	TaskStateDoing                       // 处理中
	TaskStateSuccessful                  // 成功
	TaskStateFailure
)
