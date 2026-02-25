-- ============================================================
-- 主执行脚本：按顺序执行所有数据生成脚本
-- 预计总数据量：约700万条
-- 预计执行时间：30-60分钟
-- ============================================================

SELECT '=====================================================' as message;
SELECT '开始执行百万条测试数据生成脚本' as message;
SELECT CONCAT('开始时间: ', NOW()) as message;
SELECT '=====================================================' as message;

-- 执行各层脚本
SOURCE d:/GO/workspace/goWork/work/piaowu/scripts/generate_base_data.sql;
SOURCE d:/GO/workspace/goWork/work/piaowu/scripts/generate_users.sql;
SOURCE d:/GO/workspace/goWork/work/piaowu/scripts/generate_conversations.sql;
SOURCE d:/GO/workspace/goWork/work/piaowu/scripts/generate_schedule.sql;
SOURCE d:/GO/workspace/goWork/work/piaowu/scripts/generate_ai_data.sql;

-- 验证数据生成结果
SELECT '=====================================================' as message;
SELECT '数据生成完成，验证结果：' as message;
SELECT '=====================================================' as message;

SELECT 
    'sys_roles' as table_name, 
    COUNT(*) as row_count,
    '角色表' as description
FROM sys_roles
UNION ALL
SELECT 't_shift_config', COUNT(*), '班次配置' FROM t_shift_config
UNION ALL
SELECT 't_msg_category', COUNT(*), '消息分类' FROM t_msg_category
UNION ALL
SELECT 't_conv_category', COUNT(*), '会话分类' FROM t_conv_category
UNION ALL
SELECT 't_conv_tag', COUNT(*), '会话标签' FROM t_conv_tag
UNION ALL
SELECT 't_quick_reply', COUNT(*), '快捷回复' FROM t_quick_reply
UNION ALL
SELECT 'sys_users', COUNT(*), '系统用户' FROM sys_users
UNION ALL
SELECT 't_customer_service', COUNT(*), '客服' FROM t_customer_service
UNION ALL
SELECT 't_conversation', COUNT(*), '会话' FROM t_conversation
UNION ALL
SELECT 't_conv_message', COUNT(*), '消息' FROM t_conv_message
UNION ALL
SELECT 't_conv_transfer', COUNT(*), '转接记录' FROM t_conv_transfer
UNION ALL
SELECT 't_schedule', COUNT(*), '排班' FROM t_schedule
UNION ALL
SELECT 't_leave_transfer', COUNT(*), '请假/调班' FROM t_leave_transfer
UNION ALL
SELECT 't_leave_audit_log', COUNT(*), '审批日志' FROM t_leave_audit_log
UNION ALL
SELECT 'ai_jobs', COUNT(*), 'AI任务' FROM ai_jobs
UNION ALL
SELECT 'conversation_summaries', COUNT(*), '会话摘要' FROM conversation_summaries;

SELECT '=====================================================' as message;
SELECT CONCAT('完成时间: ', NOW()) as message;
SELECT '数据生成全部完成！' as final_message;
SELECT '=====================================================' as message;
