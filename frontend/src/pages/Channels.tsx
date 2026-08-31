import { useMemo, useState } from 'react'
import {
  App,
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Switch,
  Tag,
} from 'antd'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  PlusOutlined,
  SendOutlined,
} from '@ant-design/icons'
import useSWR from 'swr'
import { channelTypes, createChannel, deleteChannel, listChannels, testChannel, updateChannel } from '../api/channel'
import type { Channel } from '../types'
import { formatTime } from '../utils/format'

const TYPE_COLORS: Record<string, string> = {
  smtp: 'blue',
  webhook: 'geekblue',
  feishu: 'cyan',
  wecom: 'green',
  dingtalk: 'orange',
  serverchan: 'purple',
}

// ChannelFormFields 按渠道类型动态渲染配置字段。
function ChannelFormFields({ type }: { type: string }) {
  switch (type) {
    case 'smtp':
      return (
        <>
          <Form.Item name={['config', 'host']} label="SMTP 服务器" rules={[{ required: true, message: '请输入服务器' }]}>
            <Input placeholder="如 smtp.qq.com" />
          </Form.Item>
          <Row gutter={12}>
            <Col span={8}>
              <Form.Item name={['config', 'port']} label="端口" rules={[{ required: true }]} initialValue={465}>
                <InputNumber style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={16}>
              <Form.Item name={['config', 'user']} label="账号" rules={[{ required: true, message: '请输入邮箱账号' }]}>
                <Input placeholder="发件邮箱账号" />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name={['config', 'password']} label="授权码/密码">
            <Input.Password placeholder="留空=不修改" />
          </Form.Item>
          <Form.Item name={['config', 'from']} label="发件人" rules={[{ required: true, message: '请输入发件人' }]}>
            <Input placeholder="发件人邮箱" />
          </Form.Item>
          <Form.Item name={['config', 'to']} label="收件人（多个用英文逗号分隔）" rules={[{ required: true }]}>
            <Input placeholder="a@x.com,b@x.com" />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name={['config', 'tls']} label="SSL 直连（465）" valuePropName="checked" initialValue>
                <Switch />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name={['config', 'insecure_skip_verify']} label="忽略证书校验" valuePropName="checked" initialValue={false}>
                <Switch />
              </Form.Item>
            </Col>
          </Row>
        </>
      )
    case 'webhook':
      return (
        <>
          <Form.Item name={['config', 'url']} label="Webhook URL" rules={[{ required: true, message: '请输入 URL' }]}>
            <Input placeholder="https://example.com/hook" />
          </Form.Item>
          <Form.Item name={['config', 'method']} label="请求方法" initialValue="POST">
            <Select options={[{ value: 'POST' }, { value: 'GET' }, { value: 'PUT' }]} />
          </Form.Item>
          <Form.Item
            name={['config', 'body_template']}
            label="请求体模板（Go text/template）"
            extra="可用 .Title .Content .Severity .Time"
          >
            <Input.TextArea rows={3} />
          </Form.Item>
        </>
      )
    case 'feishu':
    case 'dingtalk':
      return (
        <>
          <Form.Item
            name={['config', 'webhook_url']}
            label={type === 'feishu' ? '飞书机器人 Webhook' : '钉钉机器人 Webhook'}
            rules={[{ required: true, message: '请输入 Webhook' }]}
          >
            <Input placeholder="https://open.feishu.cn/... 或 https://oapi.dingtalk.com/robot/send..." />
          </Form.Item>
          <Form.Item name={['config', 'secret']} label="加签密钥（可选）" extra="留空=不加签">
            <Input.Password />
          </Form.Item>
        </>
      )
    case 'wecom':
      return (
        <>
          <Form.Item
            name={['config', 'webhook_url']}
            label="企业微信机器人 Webhook"
            rules={[{ required: true, message: '请输入 Webhook' }]}
          >
            <Input placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
          </Form.Item>
        </>
      )
    case 'serverchan':
      return (
        <>
          <Form.Item name={['config', 'sendkey']} label="Server酱 SendKey">
            <Input.Password placeholder="sctp...t" />
          </Form.Item>
        </>
      )
    default:
      return null
  }
}

// Channels 通知渠道管理页。
export default function Channels() {
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<Channel | null>(null)
  const [testingId, setTestingId] = useState<number | null>(null)
  const [form] = Form.useForm()
  const { message } = App.useApp()

  const { data: channels, isLoading, mutate } = useSWR('/channels', listChannels)
  const { data: types } = useSWR('/channels/types', channelTypes)
  const typeValue = Form.useWatch('type', form)

  const typeName = useMemo(
    () => (v?: string) => types?.find((t) => t.type === v)?.name ?? v ?? '',
    [types],
  )

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (ch: Channel) => {
    setEditing(ch)
    const cfg = { ...(ch.config ?? {}) }
    if (ch.type === 'smtp' && Array.isArray(cfg.to)) {
      cfg.to = (cfg.to as string[]).join(',')
    }
    form.setFieldsValue({ name: ch.name, type: ch.type, enabled: ch.enabled, config: cfg })
    setModalOpen(true)
  }

  const onSubmit = async () => {
    const values = await form.validateFields()
    const config = { ...values.config }
    if (values.type === 'smtp' && typeof config.to === 'string') {
      config.to = (config.to as string).split(',').map((s: string) => s.trim()).filter(Boolean)
    }
    try {
      if (editing) {
        await updateChannel(editing.id, { ...values, config })
        message.success('渠道已更新')
      } else {
        await createChannel({ ...values, config })
        message.success('渠道已创建')
      }
      setModalOpen(false)
      mutate()
    } catch (e: any) {
      message.error(e.message || '保存失败')
    }
  }

  const onTest = async (ch: Channel) => {
    setTestingId(ch.id)
    try {
      await testChannel(ch.id)
      message.success(`「${ch.name}」测试发送成功`)
    } catch (e: any) {
      message.error(`「${ch.name}」测试发送失败：${e.message || ''}`)
    } finally {
      setTestingId(null)
    }
  }

  const onDelete = async (ch: Channel) => {
    try {
      await deleteChannel(ch.id)
      message.success('已删除')
      mutate()
    } catch (e: any) {
      message.error(e.message || '删除失败')
    }
  }

  return (
    <Card
      title="通知渠道"
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建渠道
        </Button>
      }
    >
      <Row gutter={[16, 16]}>
        {(channels ?? []).map((ch) => (
          <Col xs={24} md={12} xl={8} key={ch.id}>
            <Card
              size="small"
              title={
                <Space>
                  <Tag color={TYPE_COLORS[ch.type] ?? 'default'}>{typeName(ch.type)}</Tag>
                  {ch.name}
                </Space>
              }
              extra={
                <Switch
                  size="small"
                  checked={ch.enabled}
                  onChange={async (v) => {
                    await updateChannel(ch.id, { ...ch, config: ch.config, enabled: v })
                    mutate()
                  }}
                />
              }
            >
              <div style={{ color: '#999', fontSize: 12, marginBottom: 8 }}>
                创建于 {formatTime(ch.created_at)}
              </div>
              <Space>
                <Button
                  size="small"
                  icon={<SendOutlined />}
                  loading={testingId === ch.id}
                  onClick={() => onTest(ch)}
                >
                  测试发送
                </Button>
                <Button size="small" onClick={() => openEdit(ch)}>
                  编辑
                </Button>
                <Popconfirm title="确定删除该渠道？" onConfirm={() => onDelete(ch)}>
                  <Button size="small" danger>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
              <div style={{ marginTop: 8, fontSize: 12 }}>
                {ch.enabled ? (
                  <span style={{ color: '#52c41a' }}>
                    <CheckCircleOutlined /> 已启用
                  </span>
                ) : (
                  <span style={{ color: '#999' }}>
                    <CloseCircleOutlined /> 已停用
                  </span>
                )}
              </div>
            </Card>
          </Col>
        ))}
        {(channels ?? []).length === 0 && !isLoading && (
          <Col span={24}>
            <div style={{ textAlign: 'center', color: '#999', padding: 40 }}>
              暂无通知渠道，点击右上角「新建渠道」创建
            </div>
          </Col>
        )}
      </Row>

      <Modal
        title={editing ? '编辑渠道' : '新建渠道'}
        open={modalOpen}
        onOk={onSubmit}
        onCancel={() => setModalOpen(false)}
        destroyOnHidden
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="渠道名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：运维告警邮箱" />
          </Form.Item>
          <Form.Item name="type" label="渠道类型" rules={[{ required: true, message: '请选择类型' }]}>
            <Select
              options={types?.map((t) => ({ value: t.type, label: t.name }))}
              placeholder="选择通知渠道类型"
              onChange={() => {
                // 切换类型时清空 config
                form.setFieldValue('config', {})
              }}
            />
          </Form.Item>
          {typeValue && <ChannelFormFields type={typeValue} />}
          <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
