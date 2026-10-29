# notice-scheduling Specification

## Purpose
定时通知必须在预定时间自动变为已发布，且无论扫描重复还是多实例运行都只发布一次，失败不影响其它通知。

## Requirements

### Requirement: 到点自动发布
系统 SHALL 周期性把 `status=1` 且 `scheduled_at` 不晚于当前时间的通知置为已发布（status=2）并写入发布时间。

#### Scenario: 到点通知被发布
- **WHEN** 一条定时通知的预定时间已到
- **THEN** 下一轮扫描后该通知状态为已发布且 `published_at` 非空
- **AND** 覆盖用例 `TC-NEW-SCH-0001`

#### Scenario: 未到点不发布
- **WHEN** 通知的预定时间仍在未来
- **THEN** 扫描不改变其状态
- **AND** 覆盖用例 `TC-NEW-SCH-0002`

### Requirement: 发布幂等
同一条通知 SHALL 只被发布一次；重复扫描或并发扫描不得产生重复发布时间或状态回退。

#### Scenario: 连续两轮扫描
- **WHEN** 一条到点通知在第一轮已被发布，第二轮再次被扫到
- **THEN** 第二轮不改变状态与 `published_at`
- **AND** 覆盖用例 `TC-NEW-SCH-0003`

#### Scenario: 两个实例同时扫描
- **WHEN** 两个 worker 同时处理同一条到点通知
- **THEN** 只有一个成功发布，另一个不做任何变更
- **AND** 覆盖用例 `TC-NEW-SCH-0004`

### Requirement: 失败隔离与重试
单条通知发布失败 SHALL 不影响同批其它通知，且失败通知在下一轮继续尝试。

#### Scenario: 一条失败其余成功
- **WHEN** 同批中一条通知因异常发布失败
- **THEN** 其它到点通知仍被正常发布，失败条目保持定时状态并记录日志
- **AND** 覆盖用例 `TC-NEW-SCH-0005`

### Requirement: 已撤回与草稿不受影响
扫描 SHALL 只处理 `status=1` 的通知，草稿（0）、已发布（2）、已撤回（3）一律跳过。

#### Scenario: 已撤回通知
- **WHEN** 一条已撤回通知的 `scheduled_at` 已到
- **THEN** 扫描不改变其状态
- **AND** 覆盖用例 `TC-NEW-SCH-0006`
