import { useState } from 'react'
import { App, Button, Card, Drawer, Space, Table, Tag } from 'antd'
import useSWR from 'swr'
import { ackAlert, listAlerts } from '../api/alert'
import { useAlertRealtime } from '../hooks/useRealtime'
import type { AlertEvent } from '../types'
import { formatTime } from '../utils/format'

const SEVERITY_MAP: Record<string, { color: string; label: string }> = {
  critical: { color: 'red', label: '严重' },
  warning: { color: 'orange', label: '警告' },
}

const STATUS_MAP: Record<string, { color: string; label: string }> = {
  firing: { color: 'red', label: '触发中' },
  resolved: { color: 'green', label: '已恢复' },
}

// Alerts 告警中心：事件列表 + 实时插入 + 确认。
export default function Alerts() {
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<string>('')
  const [detail, setDetail] = useState<AlertEvent | null>(null)
  const { message } = App.useApp()

  const { data, isLoading, mutate } = useSWR(
    ['/alerts', page, status],
    () => listAlerts({ status: status || undefined, page, size: 20 }),
  )

  // 收到新告警帧时刷新列表
  useAlertRealtime(() => mutate())

  const onAck = async (ev: AlertEvent) => {
    try {
      await ackAlert(ev.id)
      message.success('已确认')
      mutate()
    } catch (e: any) {
      message.error(e.message || '操作失败')
    }
  }

  const columns = [
    {
      title: '触发时间',
      dataIndex: 'fired_at',
      key: 'fired_at',
      width: 160,
      render: (v: number) => formatTime(v),
    },
    { title: '规则', dataIndex: 'rule_name', key: 'rule_name', width: 160 },
    { title: '实例', dataIndex: 'target', key: 'target', width: 130, render: (v: string) => v || '—' },
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
      title: '当前值',
      dataIndex: 'value',
      key: 'value',
      width: 90,
      render: (v: number, r: AlertEvent) => `${v?.toFixed?.(2) ?? v} / ${r.threshold}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 90,
      render: (v: string) => {
        const s = STATUS_MAP[v] ?? { color: 'default', label: v }
        return <Tag color={s.color}>{s.label}</Tag>
      },
    },
    {
      title: '持续时长',
      dataIndex: 'duration_sec',
      key: 'duration_sec',
      width: 110,
      render: (v: number, r: AlertEvent) => (r.status === 'resolved' ? `${v}s` : '—'),
    },
    { title: '已通知', dataIndex: 'notified', key: 'notified', width: 80, render: (v: boolean) => (v ? '是' : '否') },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: any, r: AlertEvent) => (
        <Space>
          <Button size="small" onClick={() => setDetail(r)}>
            详情
          </Button>
          {!r.acked && r.status === 'firing' && (
            <Button size="small" type="primary" ghost onClick={() => onAck(r)}>
              确认
            </Button>
          )}
        </Space>
      ),
    },
  ]

  return (
    <Card
      title="告警中心"
      extra={
        <Space>
          <Button
            size="small"
            type={status === '' ? 'primary' : 'default'}
            onClick={() => setStatus('')}
          >
            全部
          </Button>
          <Button
            size="small"
            type={status === 'firing' ? 'primary' : 'default'}
            danger={status === 'firing'}
            onClick={() => setStatus('firing')}
          >
            触发中
          </Button>
          <Button
            size="small"
            type={status === 'resolved' ? 'primary' : 'default'}
            onClick={() => setStatus('resolved')}
          >
            已恢复
          </Button>
        </Space>
      }
    >
      <Table<AlertEvent>
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

      <Drawer
        title="告警详情"
        open={!!detail}
        onClose={() => setDetail(null)}
        width={480}
      >
        {detail && (
          <div style={{ display: 'grid', gap: 12 }}>
            <div>
              <b>描述：</b>
              {detail.message}
            </div>
            <div>
              <b>规则：</b>
              {detail.rule_name}（id={detail.rule_id}）
            </div>
            <div>
              <b>指标：</b>
              {detail.metric} <b>实例：</b>
              {detail.target || '—'}
            </div>
            <div>
              <b>级别：</b>
              {detail.severity} <b>状态：</b>
              {detail.status}
            </div>
            <div>
              <b>当前值：</b>
              {detail.value?.toFixed?.(2) ?? detail.value} <b>阈值：</b>
              {detail.threshold}
            </div>
            <div>
              <b>触发时间：</b>
              {formatTime(detail.fired_at)}
            </div>
            {detail.resolved_at > 0 && (
              <div>
                <b>恢复时间：</b>
                {formatTime(detail.resolved_at)}（持续 {detail.duration_sec}s）
              </div>
            )}
            <div>
              <b>已通知：</b>
              {detail.notified ? `是（${formatTime(detail.notify_at)}）` : '否'}
            </div>
            <div>
              <b>已确认：</b>
              {detail.acked ? `${detail.ack_by}（${formatTime(detail.ack_at)}）` : '否'}
            </div>
          </div>
        )}
      </Drawer>
    </Card>
  )
}
