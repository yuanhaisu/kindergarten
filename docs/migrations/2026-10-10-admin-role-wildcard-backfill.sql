-- wire-admin-permissions: 给系统角色「园所管理员」回填通配权限（*）
--
-- 背景：管理端权限细闸上线后，未配任何权限码的角色将失去全部管理端访问。
-- 系统角色「园所管理员」（is_system=1）当前权限为空，属播种缺口而非产品意图——
-- 回填权限目录中已有的 "*" 行，使其名下管理员保持与强制前等价的访问；
-- 园长可随后在角色页把 "*" 改为显式码集来收紧。
--
-- 幂等：唯一键 idx_role_perm (role_id, permission_id) 冲突保护，可反复执行。
-- 语义对齐 ReplaceRolePermissionLinks：关联行按硬删管理，冲突检查不过滤 deleted_at。
-- 前置：admin_permissions 目录已含 code='*' 的行（service 启动时自动播种）。

INSERT INTO admin_role_permissions (role_id, permission_id, created_at, updated_at)
SELECT ar.id, ap.id, NOW(), NOW()
FROM admin_roles ar
JOIN admin_permissions ap ON ap.code = '*' AND ap.deleted_at IS NULL
WHERE ar.is_system = 1
  AND ar.name = '园所管理员'
  AND ar.deleted_at IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM admin_role_permissions arp
    WHERE arp.role_id = ar.id AND arp.permission_id = ap.id
  );

-- 对账（每个系统「园所管理员」角色都应查到一行 code='*'）：
-- SELECT ar.id AS role_id, ar.kindergarten_id, ar.name, ap.code
-- FROM admin_roles ar
--   JOIN admin_role_permissions arp ON arp.role_id = ar.id
--   JOIN admin_permissions ap ON ap.id = arp.permission_id
-- WHERE ar.is_system = 1 AND ar.name = '园所管理员' AND ar.deleted_at IS NULL;
