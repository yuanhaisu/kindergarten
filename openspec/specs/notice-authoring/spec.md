# notice-authoring Specification

## Purpose
老师端通知作者需要能保存不完整的草稿、预约定时发送、附带图片并在发布前继续编辑，同时保证草稿只对作者可见、已发布通知不可再改。

## Requirements

### Requirement: 老师可保存不完整草稿
老师端 SHALL 允许把通知保存为草稿（status=0），此时标题、正文、目标班级可以为空，只要标题或正文至少一项非空。

#### Scenario: 仅填标题存草稿
- **WHEN** 老师只填写标题并点击存草稿
- **THEN** 通知以草稿状态创建成功，目标班级为空
- **AND** 覆盖用例 `TC-NEW-NOT-0001`

#### Scenario: 标题正文都为空
- **WHEN** 老师未填标题也未填正文就存草稿
- **THEN** 返回参数错误，不创建草稿
- **AND** 覆盖用例 `TC-NEW-NOT-0002`

### Requirement: 草稿仅创建者可见
草稿 SHALL 只对创建者可见；其他老师查看列表与详情时不得看到该草稿。

#### Scenario: 其他老师查看列表
- **WHEN** 同班另一位老师读取通知列表（不传 status）
- **THEN** 结果不包含他人草稿
- **AND** 覆盖用例 `TC-NEW-NOT-0003`

#### Scenario: 其他老师直接打开详情
- **WHEN** 其他老师用草稿 id 请求详情
- **THEN** 返回无权限/不存在，不返回内容
- **AND** 覆盖用例 `TC-NEW-NOT-0004`

### Requirement: 定时发送必须给出未来时间
老师提交定时通知 SHALL 提供带时区的 RFC3339 时间且晚于当前时间；缺失、格式错误或已过期都返回参数错误。

#### Scenario: 提交过去时间
- **WHEN** 老师提交 status=1 且 scheduled_at 早于当前时间
- **THEN** 返回参数错误，不创建通知
- **AND** 覆盖用例 `TC-NEW-NOT-0005`

#### Scenario: 正确时区格式
- **WHEN** 老师提交 `2026-11-01T09:00:00+08:00`
- **THEN** 通知以定时状态保存，时间按同一时区解释
- **AND** 覆盖用例 `TC-NEW-NOT-0006`

### Requirement: 仅草稿与定时可编辑
系统 SHALL 只允许编辑 status=0 或 status=1 的通知；已发布或已撤回的通知编辑时返回业务错误。

#### Scenario: 编辑已发布通知
- **WHEN** 老师尝试编辑 status=2 的通知
- **THEN** 返回业务错误，通知内容不变
- **AND** 覆盖用例 `TC-NEW-NOT-0007`

### Requirement: 发布草稿要求内容完整
把草稿发布（status=2）时 SHALL 校验标题、正文、目标班级均非空，并写入发布时间；不完整则拒绝发布且保留草稿。

#### Scenario: 不完整草稿直接发布
- **WHEN** 老师对缺少正文的草稿点击发布
- **THEN** 返回参数错误，草稿保持不变
- **AND** 覆盖用例 `TC-NEW-NOT-0008`

### Requirement: 附件仅限图片且有数量上限
通知附件 SHALL 只接受图片类型，最多 9 张，单张不超过 20MB；创建与编辑时保存，详情返回附件列表，列表返回附件数量。

#### Scenario: 上传 3 张图片
- **WHEN** 老师创建通知并附带 3 张合法图片
- **THEN** 详情返回 3 条附件，列表 attachments_count 为 3
- **AND** 覆盖用例 `TC-NEW-NOT-0009`

#### Scenario: 超过 9 张
- **WHEN** 老师一次提交 10 张图片
- **THEN** 返回参数错误，通知不保存
- **AND** 覆盖用例 `TC-NEW-NOT-0010`

### Requirement: 删除与撤回规则
系统 SHALL 只允许删除草稿；已发布通知只能走撤回，撤回后状态为 3。

#### Scenario: 删除非草稿
- **WHEN** 老师删除一条已发布通知
- **THEN** 返回业务错误，通知仍存在
- **AND** 覆盖用例 `TC-NEW-NOT-0011`

### Requirement: 家长只能确认已发布通知
家长端确认通知 SHALL 仅对 status=2 的通知生效；草稿、定时、已撤回通知不得被确认。

#### Scenario: 确认定时通知
- **WHEN** 家长用定时通知 id 调用确认
- **THEN** 返回业务错误，不产生确认记录
- **AND** 覆盖用例 `TC-NEW-NOT-0012`
