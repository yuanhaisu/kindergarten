# class-album-notify Specification

## Purpose
班级相册发布成功后，向相册参与者的家长写一条站内消息，让家长从消息中心直接进入该相册详情；不改变照片可见性规则，也不做微信推送。

## Requirements

### Requirement: 发布成功后写站内消息

相册从草稿或已下架成功发布后，系统 SHALL 在发布事务外为符合条件的家长各写一条站内消息；消息写入失败 MUST NOT 导致发布失败或回滚。

#### Scenario: 发布成功产生消息
- **WHEN** 老师发布一个含参与者与照片的相册
- **THEN** 每位符合条件的家长收到一条 `type=album` 的站内消息，`reference_id` 为相册 ID，标题与内容包含相册标题
- **AND** 覆盖用例 `TC-NEW-ALBMSG-0001`

#### Scenario: 通知写入失败不影响发布
- **WHEN** 消息仓库不可用（或未接线）
- **THEN** 发布仍返回成功，相册状态为已发布，仅记录错误日志
- **AND** 覆盖用例 `TC-NEW-ALBMSG-0002`

### Requirement: 收件人范围与班内共享过滤

收件人 MUST 只包含相册参与者的监护人；监护人本人 `allow_class_share=false` 时 MUST NOT 收到消息；无设置行时按默认 `true` 处理；MUST NOT 向非参与者家庭发送。

#### Scenario: opt-out 家长不收
- **WHEN** 参与幼儿 C 的监护人已关闭班内共享
- **THEN** 该监护人不收到消息
- **AND** 覆盖用例 `TC-NEW-ALBMSG-0003`

#### Scenario: 同一孩子另一位监护人默认收到
- **WHEN** 参与幼儿 C 的另一位监护人未关闭班内共享（无设置行）
- **THEN** 该监护人收到消息；同一家长关联多个参与孩子时只收到一条
- **AND** 覆盖用例 `TC-NEW-ALBMSG-0004`

#### Scenario: 非参与者不收
- **WHEN** 某家长的孩子不在相册参与者中
- **THEN** 该家长不收到任何该相册的消息
- **AND** 覆盖用例 `TC-NEW-ALBMSG-0005`

### Requirement: 同一相册不重复发送

重复发布、下架后重新发布，或老师修改相册标题后重发，同一家长对同一相册 MUST NOT 收到第二条消息。

#### Scenario: 重新发布不重复
- **WHEN** 相册下架后再次发布
- **THEN** 已收到过该相册消息的家长不再新增消息
- **AND** 覆盖用例 `TC-NEW-ALBMSG-0006`

### Requirement: 消息跳转与家长端渲染

消息的 `action_path` MUST 是内部 `/pages/` 路径，指向该家长参与孩子的相册详情；家长端消息中心 SHALL 识别 `album` 类型并沿用现有已读与跳转逻辑。

#### Scenario: 消息直达相册详情
- **WHEN** 家长在消息中心点击相册消息
- **THEN** 标记已读并打开 `/pages/album/detail?id=<album_id>&student_id=<孩子>`；相册不存在时沿用「该内容已不存在」提示
- **AND** 覆盖用例 `TC-NEW-ALBMSG-0007`
