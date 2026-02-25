// 统一请求封装（中文注释）：所有页面都通过这里访问网关接口

const DEFAULT_BASE_URL = 'http://localhost:8081'

export function getBaseUrl() {
	// 强制锁定8081端口，防止旧缓存干扰
	return 'http://localhost:8081'
}

export function request(options) {
	const opts = options || {}
	const path = opts.path || ''
	const method = (opts.method || 'GET').toUpperCase()
	const data = opts.data
	const headers = opts.headers || {}

	return new Promise((resolve, reject) => {
		const url = getBaseUrl() + path
		const token = uni.getStorageSync('token')

		const header = Object.assign(
			{
				'Content-Type': 'application/json'
			},
			headers
		)
		if (typeof token === 'string' && token.trim()) {
			header['Authorization'] = 'Bearer ' + token.trim()
		}

		uni.request({
			url,
			method,
			data,
			header,
			timeout: 15000,
			success: (res) => {
				// 兼容处理：部分环境在非 2xx 时不自动 JSON 反序列化，导致 res.data 变成字符串
				if (res && typeof res.data === 'string') {
					const text = res.data.trim()
					if (text && (text[0] === '{' || text[0] === '[')) {
						try {
							res.data = JSON.parse(text)
						} catch (e) { }
					}
				}
				resolve(res ? res.data : null)
			},
			fail: (err) => reject(err)
		})
	})
}

// 二进制请求：用于导出 Excel 等下载场景
export function requestArrayBuffer(options) {
	const opts = options || {}
	const path = opts.path || ''
	const method = (opts.method || 'GET').toUpperCase()
	const data = opts.data
	const headers = opts.headers || {}

	return new Promise((resolve, reject) => {
		const url = getBaseUrl() + path
		const token = uni.getStorageSync('token')

		const header = Object.assign({}, headers)
		if (typeof token === 'string' && token.trim()) {
			header['Authorization'] = 'Bearer ' + token.trim()
		}

		uni.request({
			url,
			method,
			data,
			header,
			timeout: 60000,
			responseType: 'arraybuffer',
			success: (res) => resolve(res || null),
			fail: (err) => reject(err)
		})
	})
}

// 解析后端统一响应（Thrift JSON）：优先读取 base_resp.code/msg
export function getBaseResp(body) {
	if (!body) {
		return { code: -1, msg: '空响应' }
	}
	if (typeof body === 'string') {
		const text = body.trim()
		if (text && (text[0] === '{' || text[0] === '[')) {
			try {
				body = JSON.parse(text)
			} catch (e) {
				return { code: -1, msg: text || '未知响应结构' }
			}
		} else {
			return { code: -1, msg: text || '未知响应结构' }
		}
	}
	// 1. 标准结构: base_resp.code
	const base = body.base_resp || body.BaseResp
	if (base && typeof base.code === 'number') {
		return { code: base.code, msg: base.msg || '' }
	}
	// 2. 扁平结构: body.code
	if (typeof body.code === 'number') {
		return { code: body.code, msg: body.msg || '' }
	}
	// 3. Kitex/框架异常结构: {"result": -2147483572, "msg": "..."}
	if (typeof body.result === 'number') {
		return {
			code: body.result,
			msg: body.msg || (body.result < 0 ? '系统繁忙或RPC调用异常' : '')
		}
	}
	return { code: -1, msg: '未知响应结构' }
}

export function ensureOk(body) {
	const base = getBaseResp(body)
	if (base.code === 0) return
	const e = new Error(base.msg || '请求失败')
	e.code = base.code
	e.body = body
	throw e
}
