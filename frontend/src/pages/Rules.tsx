import { useMemo, useState } from 'react'
import {
  App,
  Button,
  Card,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tag,
} from 'antd'
import { PlusOutlined, ReloadOutlined } from '@ant-design/icons'
import useSWR from 'swr'
import { createRule, deleteRule, listRules, reloadRules, toggleRule, updateRule } from '../api/rule'
import { listChannels } from '../api/channel'
import type { AlertRule } from '../types'
import { formatTime } from '../utils/format'

const METRIC_OPTIONS = [
  { value: 'cpu_usage', label: 'CPU 使用率（%）', hint: '无需目标' },
  { value: 'mem_usage', label: '内存使用率（%）', hint: '无需目标' },
  { value: 'load1', label: '系统负载 1m', hint: '无需目标' },
  { value: 'disk_used_percent', label: '磁盘使用率（%）', hint: '挂载点如 / 或 C:，留空=所有分区' },
  { value: 'net_rx_bps', label: '网络入带宽（B/s）', hint: '网卡名，留空=全部合计' },
  { value: 'net_tx_bps', label: '网络出带宽（B/s）', hint: '网卡名，留空=全部合计' },
  { value: 'service_active', label: '服务状态（1=运行 0=停止）', hint: 'systemd 服务名，如 sshd.service' },
  { value: 'process_cpu', label: '进程 CPU（%）', hint: '进程名，如 nginx' },
]

const OP_OPTIONS = [
  { value: 'gt', label: '>' },
  { value: 'ge', label: '>=' },
  { value: 'lt', label: '<' },
  { value: 'le', label: '<=' },
]

const SEVERITY_MAP: Record<string, { color: string; label: string }> = {
  critical: { color: 'red', label: '严重' },
  warning: { color: 'orange', label: '警告' },
}

// Rules 告警规则管理页。
export default function Rules() {
  const [page, setPage] = useState(1)
  const [modalOpen, setModalOpen] = useState(false)
  const [editing, setEditing] = useState<AlertRule | null>(null)
  const [form] = Form.useForm()
  const { message } = App.useApp()

  const { data, isLoading, mutate } = useSWR(['/rules', page], () => listRules(page, 20))
  const { data: channels } = useSWR('/channels', listChannels)
  const metricValue = Form.useWatch('metric', form)

  const metricHint = useMemo(() => METRIC_OPTIONS.find((m) => m.value === metricValue)?.hint, [metricValue])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setModalOpen(true)
  }

  const openEdit = (row: AlertRule) => {
    setEditing(row)
    form.setFieldsValue({
      ...row,
      channel_ids: row.channel_ids ?? [],
    })
    setModalOpen(true)
  }

  const onSubmit = async () => {
    const values = await form.validateFields()
    try {
      if (editing) {
        await updateRule(editing.id, values)
        message.success('规则已更新')
      } else {
        await createRule(values)
        message.success('规则已创建')
      }
      setModalOpen(false)
      mutate()
    } catch (e: any) {
      message.error(e.message || '保存失败')
    }
  }

  const onToggle = async (row: AlertRule, enabled: boolean) => {
    try {
      await toggleRule(row.id, enabled)
      message.success(enabled ? '已启用' : '已停用')
      mutate()
    } catch (e: any) {
      message.error(e.message || '操作失败')
    }
  }

  const onDelete = async (row: AlertRule) => {
    try {
      await deleteRule(row.id)
      message.success('已删除')
      mutate()
    } catch (e: any) {
      message.error(e.message || '删除失败')
    }
  }

  const columns = [
    { title: '名称', dataIndex: 'name', key: 'name', width: 160 },
    {
      title: '指标',
      dataIndex: 'metric',
      key: 'metric',
      width: 160,
      render: (v: string) => METRIC_OPTIONS.find((m) => m.value === v)?.label ?? v,
    },
    { title: '目标', dataIndex: 'target', key: 'target', width: 130, render: (v: string) => v || '—' },
    {
      title: '条件',
      key: 'cond',
      width: 160,
      render: (_: any, r: AlertRule) => (
        <span>
          {r.threshold}
          <span style={{ color: '#999', marginLeft: 8 }}>连续 {r.duration_ticks} 周期</span>
        </span>
      ),
    },
    {
      title: '级别',
      dataIndex: 'severity',
      key: 'severity',
      width: 80,
      render: (v: string) => {
        const s = SEVERITY_MAP[v] ?? { color: 'default', label: v }
        return <Tag color={s.color}>{s.label}</Tag>
      },
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 90,
      render: (v: boolean, r: AlertRule) => <Switch size="small" checked={v} onChange={(e) => onToggle(r, e)} />,
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 140,
      render: (v: number) => formatTime(v),
    },
    {
      title: '操作',
      key: 'action',
      width: 130,
      render: (_: any, r: AlertRule) => (
        <Space>
          <Button size="small" onClick={() => openEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="确定删除该规则？" onConfirm={() => onDelete(r)}>
            <Button size="small" danger>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <Card
      title="告警规则"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => reloadRules().then(() => message.success('规则已重载'))}>
            重载
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建规则
          </Button>
        </Space>
      }
    >
      <Table<AlertRule>
        rowKey="id"
        loading={isLoading}
        dataSource={data?.list ?? []}
        columns={columns}
        pagination={{
          current: page,
          pageSize: 20,
          total: data?.total ?? 0,
          onChange: setPage,
          showSizeChanger: false,
          showTotal: (t) => `共 ${t} 条`,
        }}
      />

      <Modal
        title={editing ? '编辑规则' : '新建规则'}
        open={modalOpen}
        onOk={onSubmit}
        onCancel={() => setModalOpen(false)}
        destroyOnHidden
        width={560}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="如：CPU 使用率过高" />
          </Form.Item>
          <Form.Item name="metric" label="指标" rules={[{ required: true, message: '请选择指标' }]}>
            <Select options={METRIC_OPTIONS} placeholder="选择监控指标" />
          </Form.Item>
          <Form.Item name="target" label="目标实例" extra={metricHint}>
            <Input placeholder="留空表示全部实例" />
          </Form.Item>
          <Space size={12} style={{ display: 'flex' }}>
            <Form.Item name="operator" label="操作符" rules={[{ required: true }]} style={{ width: 100 }}>
              <Select options={OP_OPTIONS} />
            </Form.Item>
            <Form.Item name="threshold" label="阈值" rules={[{ required: true, message: '请输入阈值' }]} style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>
            <Form.Item name="duration_ticks" label="持续周期" initialValue={1} style={{ width: 130 }}>
              <InputNumber min={1} />
            </Form.Item>
          </Space>
          <Space size={12} style={{ display: 'flex' }}>
            <Form.Item name="severity" label="级别" initialValue="warning" style={{ width: 130 }}>
              <Select
                options={[
                  { value: 'warning', label: '警告' },
                  { value: 'critical', label: '严重' },
                ]}
              />
            </Form.Item>
            <Form.Item name="cooldown_sec" label="通知冷却（秒）" initialValue={900} style={{ width: 160 }}>
              <InputNumber min={0} />
            </Form.Item>
            <Form.Item name="notify_on_resolve" label="恢复时通知" valuePropName="checked" initialValue={false} style={{ marginTop: 6 }}>
              <Switch />
            </Form.Item>
          </Space>
          <Form.Item
            name="channel_ids"
            label="通知渠道（可多选，留空则不通知）"
            extra="在「通知渠道」页面配置"
          >
            <Select
              mode="multiple"
              allowClear
              placeholder="选择通知渠道"
              options={(channels ?? []).filter((c) => c.enabled).map((c) => ({ value: c.id, label: c.name }))}
            />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
          <Form.Item name="description" label="备注">
            <Input.TextArea rows={2} placeholder="可选" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
