# Spec Delta

## Purpose

老师需要按分类和月份管理班级照片：创建相册、查看统计、按分类筛选相册、浏览相册内大图，数据仅本班可见。

## ADDED Requirements

### Requirement: 创建班级相册
老师 SHALL 能在本班创建相册，填写标题、分类与日期，并关联已上传的照片（至少 1 张、最多 50 张）。

#### Scenario: 创建含 3 张照片的相册
- **WHEN** 老师提交标题、分类「活动」与 3 张已上传照片
- **THEN** 相册创建成功，照片按提交顺序保存
- **AND** 覆盖用例 `TC-NEW-ALB-0001`

#### Scenario: 没有照片
- **WHEN** 老师提交相册但没有照片
- **THEN** 返回参数错误，不创建相册
- **AND** 覆盖用例 `TC-NEW-ALB-0002`

### Requirement: 相册统计
系统 SHALL 返回本班照片总数、本月新增照片数与相册总数。

#### Scenario: 有跨月相册
- **WHEN** 本班有 2 个本月相册与 1 个上月相册
- **THEN** 相册数为 3，本月新增只统计本月相册的照片
- **AND** 覆盖用例 `TC-NEW-ALB-0003`

### Requirement: 按分类筛选相册
相册列表 SHALL 支持按分类筛选（活动/日常/作品/节日），不传时返回全部；结果按相册日期倒序。

#### Scenario: 筛选活动
- **WHEN** 老师选择「活动」分类
- **THEN** 只返回该班活动分类的相册，日期倒序
- **AND** 覆盖用例 `TC-NEW-ALB-0004`

### Requirement: 相册详情与照片顺序
相册详情 SHALL 返回相册元信息与照片列表，照片按创建时的排序返回。

#### Scenario: 打开相册
- **WHEN** 老师打开含 12 张照片的相册
- **THEN** 返回 12 张照片且顺序与创建时一致
- **AND** 覆盖用例 `TC-NEW-ALB-0005`

### Requirement: 班级与园所隔离
相册 SHALL 只属于创建时的班级与园所；其他班级或园所的老师不可读取或创建到该班。

#### Scenario: 别班老师读取
- **WHEN** 其他班老师请求该相册
- **THEN** 返回不存在/无权限
- **AND** 覆盖用例 `TC-NEW-ALB-0006`

### Requirement: 分类取值受限
相册分类 SHALL 只接受 `activity`、`daily`、`work`、`festival` 四值，非法值返回参数错误。

#### Scenario: 非法分类
- **WHEN** 老师提交分类 `other`
- **THEN** 返回参数错误，不创建相册
- **AND** 覆盖用例 `TC-NEW-ALB-0007`
