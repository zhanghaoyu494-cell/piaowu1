// 接口清单（中文注释）：与 gateway/router/router.go 保持一致

import { request, requestArrayBuffer } from './request.js'

function toQuery(params) {
	const p = params || {}
	const parts = []
	Object.keys(p).forEach((k) => {
		const v = p[k]
		if (v === undefined || v === null || v === '') return
		parts.push(encodeURIComponent(k) + '=' + encodeURIComponent(String(v)))
	})
	return parts.length ? '?' + parts.join('&') : ''
}

export const api = {
	// 客服基础信息
	getCustomerService(csId) {
		return request({
			path: '/api/customer/get' + toQuery({ cs_id: csId }),
			method: 'GET'
		})
	},
	listCustomerService(params) {
		return request({
			path: '/api/customer/list' + toQuery(params),
			method: 'GET'
		})
	},

	// 班次配置
	createShiftConfig(payload) {
		return request({
			path: '/api/shift/create',
			method: 'POST',
			data: payload
		})
	},
	listShiftConfig(params) {
		return request({
			path: '/api/shift/list' + toQuery(params),
			method: 'GET'
		})
	},
	// 更新班次配置
	updateShiftConfig(payload) {
		return request({
			path: '/api/shift/update',
			method: 'POST',
			data: payload
		})
	},
	// 删除班次配置
	deleteShiftConfig(shiftId) {
		return request({
			path: '/api/shift/delete',
			method: 'POST',
			data: { shift_id: shiftId }
		})
	},

	// 排班
	assignSchedule(payload) {
		return request({
			path: '/api/schedule/assign',
			method: 'POST',
			data: payload
		})
	},
	listScheduleGrid(params) {
		return request({
			path: '/api/schedule/grid' + toQuery(params),
			method: 'GET'
		})
	},
	exportScheduleExcel(params) {
		return requestArrayBuffer({
			path: '/api/schedule/export' + toQuery(params),
			method: 'GET'
		})
	},
	upsertScheduleCell(payload) {
		return request({
			path: '/api/schedule/cell/upsert',
			method: 'POST',
			data: payload
		})
	},
	autoSchedule(payload) {
		return request({
			path: '/api/schedule/auto',
			method: 'POST',
			data: payload
		})
	},

	// 请假/调班
	applyLeaveTransfer(payload) {
		return request({
			path: '/api/leave/apply',
			method: 'POST',
			data: payload
		})
	},
	approveLeaveTransfer(payload) {
		return request({
			path: '/api/leave/approve',
			method: 'POST',
			data: payload
		})
	},
	getLeaveTransfer(applyId) {
		return request({
			path: '/api/leave/get' + toQuery({ apply_id: applyId }),
			method: 'GET'
		})
	},
	listLeaveTransfer(params) {
		return request({
			path: '/api/leave/list' + toQuery(params),
			method: 'GET'
		})
	},
	// 获取请假审计日志
	getLeaveAuditLog(applyId) {
		return request({
			path: '/api/leave/audit-log' + toQuery({ apply_id: applyId }),
			method: 'GET'
		})
	},
	// 获取调班候选人
	getSwapCandidates(params) {
		return request({
			path: '/api/leave/swap-candidates' + toQuery(params),
			method: 'GET'
		})
	},
	// 检测调班冲突
	checkSwapConflict(params) {
		return request({
			path: '/api/leave/check-conflict',
			method: 'POST',
			data: params
		})
	},

	// 心跳与在线状态
	heartbeat(csId) {
		return request({
			path: '/api/customer/heartbeat',
			method: 'POST',
			data: { cs_id: csId }
		})
	},
	listOnlineCustomers(params) {
		return request({
			path: '/api/customer/online-list' + toQuery(params),
			method: 'GET'
		})
	},

	// 会话转接
	transferConversation(payload) {
		return request({
			path: '/api/conversation/transfer',
			method: 'POST',
			data: payload
		})
	},

	// 创建会话（用户发起新会话）
	// payload: { user_id, user_nickname, source, cs_id, first_msg }
	createConversation(payload) {
		return request({
			path: '/api/conversation/create',
			method: 'POST',
			data: payload
		})
	},

	// 结束会话
	// payload: { conv_id, operator_id, end_reason }
	endConversation(payload) {
		return request({
			path: '/api/conversation/end',
			method: 'POST',
			data: payload
		})
	},

	// 自动分配客服（用户发起咨询时调用）
	// payload: { user_id, user_nickname, source }
	assignCustomer(payload) {
		return request({
			path: '/api/conversation/assign',
			method: 'POST',
			data: payload
		})
	},

	// 会话管理/记录查询
	listConversation(params) {
		return request({
			path: '/api/conversation/list' + toQuery(params),
			method: 'GET'
		})
	},
	listConversationHistory(params) {
		return request({
			path: '/api/conversation/history/list' + toQuery(params),
			method: 'GET'
		})
	},
	listConversationMessage(params) {
		return request({
			path: '/api/conversation/message/list' + toQuery(params),
			method: 'GET'
		})
	},
	sendConversationMessage(payload) {
		return request({
			path: '/api/conversation/message/send',
			method: 'POST',
			data: payload
		})
	},

	// 快捷回复
	listQuickReply(params) {
		return request({
			path: '/api/quick_reply/list' + toQuery(params),
			method: 'GET'
		})
	},
	// 创建快捷回复
	// payload: { reply_type, reply_content, create_by, is_public }
	createQuickReply(payload) {
		return request({
			path: '/api/quick_reply/create',
			method: 'POST',
			data: payload
		})
	},
	// 更新快捷回复
	// payload: { reply_id, reply_type, reply_content, is_public }
	updateQuickReply(payload) {
		return request({
			path: '/api/quick_reply/update',
			method: 'POST',
			data: payload
		})
	},
	// 删除快捷回复
	deleteQuickReply(replyId) {
		return request({
			path: '/api/quick_reply/delete',
			method: 'POST',
			data: { reply_id: replyId }
		})
	},

	// 会话分类
	createConvCategory(payload) {
		return request({
			path: '/api/conversation/category/create',
			method: 'POST',
			data: payload
		})
	},
	listConvCategory(params) {
		return request({
			path: '/api/conversation/category/list' + toQuery(params),
			method: 'GET'
		})
	},
	updateConversationClassify(payload) {
		return request({
			path: '/api/conversation/classify/update',
			method: 'POST',
			data: payload
		})
	},

	// ============ 会话标签管理 ============
	// 创建标签
	// payload: { tag_name, tag_color, sort_no, create_by }
	createConvTag(payload) {
		return request({
			path: '/api/conversation/tag/create',
			method: 'POST',
			data: payload
		})
	},
	// 查询标签列表
	listConvTag(params) {
		return request({
			path: '/api/conversation/tag/list' + toQuery(params),
			method: 'GET'
		})
	},
	// 更新标签
	// payload: { tag_id, tag_name, tag_color, sort_no }
	updateConvTag(payload) {
		return request({
			path: '/api/conversation/tag/update',
			method: 'POST',
			data: payload
		})
	},
	// 删除标签
	deleteConvTag(tagId) {
		return request({
			path: '/api/conversation/tag/delete',
			method: 'POST',
			data: { tag_id: tagId }
		})
	},

	// ============ 统计看板 ============
	// 获取会话统计数据
	// params: { start_date, end_date, stat_type }
	// stat_type: day/week/month
	getConversationStats(params) {
		return request({
			path: '/api/conversation/stats' + toQuery(params),
			method: 'GET'
		})
	},
	// 获取在线状态统计（当前在线客服和连接数）
	getOnlineStats() {
		return request({
			path: '/api/stats/online',
			method: 'GET'
		})
	},

	// ============ 会话监控与导出 ============
	// 获取会话监控数据（实时）
	// params: { dept_id, status_filter }
	// status_filter: -1-全部 0-等待 1-进行中
	getConversationMonitor(params) {
		return request({
			path: '/api/conversation/monitor' + toQuery(params),
			method: 'GET'
		})
	},
	// 导出会话记录
	// params: { cs_id, user_id, start_date, end_date, status, keyword, export_format }
	// export_format: excel/csv
	exportConversations(params) {
		return requestArrayBuffer({
			path: '/api/conversation/export' + toQuery(params),
			method: 'GET'
		})
	},

	// ============ 消息分类管理 ============
	// 消息自动分类
	// payload: { conv_id, msg_contents }
	msgAutoClassify(payload) {
		return request({
			path: '/api/msg/classify/auto',
			method: 'POST',
			data: payload
		})
	},
	// 人工调整分类
	// payload: { conv_id, original_category_id, new_category_id, operator_id, adjust_reason }
	adjustMsgClassify(payload) {
		return request({
			path: '/api/msg/classify/adjust',
			method: 'POST',
			data: payload
		})
	},
	// 获取分类统计数据
	// params: { start_date, end_date, stat_type }
	getClassifyStats(params) {
		return request({
			path: '/api/msg/classify/stats' + toQuery(params),
			method: 'GET'
		})
	},

	// ============ 消息分类维度管理 ============
	// 创建消息分类维度
	// payload: { category_name, keywords, sort_no, create_by }
	createMsgCategory(payload) {
		return request({
			path: '/api/msg/category/create',
			method: 'POST',
			data: payload
		})
	},
	// 查询消息分类维度列表
	listMsgCategory() {
		return request({
			path: '/api/msg/category/list',
			method: 'GET'
		})
	},
	// 更新消息分类维度
	// payload: { category_id, category_name, keywords, sort_no }
	updateMsgCategory(payload) {
		return request({
			path: '/api/msg/category/update',
			method: 'POST',
			data: payload
		})
	},
	// 删除消息分类维度
	deleteMsgCategory(categoryId) {
		return request({
			path: '/api/msg/category/delete',
			method: 'POST',
			data: { category_id: categoryId }
		})
	},

	// ============ 消息加密与脱敏 ============
	// 加密消息内容
	// payload: { msg_content }
	encryptMessage(payload) {
		return request({
			path: '/api/msg/encrypt',
			method: 'POST',
			data: payload
		})
	},
	// 解密消息内容
	// payload: { encrypted_content }
	decryptMessage(payload) {
		return request({
			path: '/api/msg/decrypt',
			method: 'POST',
			data: payload
		})
	},
	// 消息脱敏处理
	// payload: { msg_content }
	desensitizeMessage(payload) {
		return request({
			path: '/api/msg/desensitize',
			method: 'POST',
			data: payload
		})
	},

	// ============ 数据归档管理 ============
	// 归档历史会话
	// payload: { end_date, retention_days, operator_id }
	archiveConversations(payload) {
		return request({
			path: '/api/archive/conversations',
			method: 'POST',
			data: payload
		})
	},
	// 获取归档任务状态
	// params: { task_id }
	getArchiveTask(taskId) {
		return request({
			path: '/api/archive/task' + toQuery({ task_id: taskId }),
			method: 'GET'
		})
	},
	// 查询归档会话
	// params: { user_id, cs_id, start_date, end_date, page, page_size }
	queryArchivedConversation(params) {
		return request({
			path: '/api/archive/query' + toQuery(params),
			method: 'GET'
		})
	},

	// ============ 用户认证 ============
	// 用户登录
	login(userName, password) {
		return request({
			path: '/api/v1/user/login',
			method: 'POST',
			data: { user_name: userName, password: password }
		})
	},
	// 用户注册（仅能注册客服账号）
	register(userName, password, realName, phone = '') {
		return request({
			path: '/api/v1/user/register',
			method: 'POST',
			data: { user_name: userName, password: password, real_name: realName, phone: phone }
		})
	},
	// 获取当前用户信息
	getCurrentUser() {
		return request({
			path: '/api/v1/user/current',
			method: 'GET'
		})
	},
	// 退出登录（调用后端接口置offline，并清除本地存储）
	logout(csId) {
		// 异步调用后端logout接口
		if (csId) {
			request({
				path: '/api/v1/user/logout',
				method: 'POST',
				data: { cs_id: csId }
			}).catch(e => console.error('logout api error:', e))
		}
		// 清除本地存储
		uni.removeStorageSync('token')
		uni.removeStorageSync('userInfo')
		uni.removeStorageSync('isLogin')
		uni.removeStorageSync('roleCode')
	}
}