import { useState } from 'react'
import { Badge, Card, Drawer, Space, Table, Tag } from 'antd'
import useSWR from 'swr'
import { listServices, serviceHistory } from '../api/service'
import type { ServiceState } from '../types'
import { formatTime } from '../utils/format'

type BadgeStatus = 'success' | 'default' | 'error' | 'processing' | 'warning'

const ACTIVE_COLOR: Record<string, BadgeStatus> = {
  active: 'success',
  inactive: 'default',
  failed: 'error',
  activating: 'processing',
}

const SUB_STATE_LABEL: Record<string, string> = {
  running: '运行中',
  exited: '已退出',
  dead: '已停止',
  failed: '失败',
  'auto-restart': '自动重启',
  start: '启动中',
}

// Service 服务监控页：systemd 服务状态列表。
export default function Service() {
  const [state, setState] = useState('')
  const [detail, setDetail] = useState<string | null>(null)

  const { data, isLoading } = useSWR(['/services', state], () => listServices(state || undefined), {
    refreshInterval: 30000,
  })

  const activeCount = (data ?? []).filter((s) => s.active_state === 'active').length
  const failedCount = (data ?? []).filter((s) => s.active_state === 'failed').length

  const columns = [
    { title: '服务', dataIndex: 'name', key: 'name', width: 200 },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true, render: (v: string) => v || '—' },
    {
      title: '状态',
      dataIndex: 'active_state',
      key: 'active_state',
      width: 90,
      render: (v: string) => (
        <Badge status={ACTIVE_COLOR[v] ?? 'default'} text={v === 'active' ? '运行中' : v} />
      ),
    },
    {
      title: '子状态',
      dataIndex: 'sub_state',
      key: 'sub_state',
      width: 110,
      render: (v: string) => <Tag color={v === 'running' ? 'green' : 'default'}>{SUB_STATE_LABEL[v] ?? v}</Tag>,
    },
    { title: '主 PID', dataIndex: 'main_pid', key: 'main_pid', width: 90, render: (v: number) => v || '—' },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_: any, r: ServiceState) => (
        <a onClick={() => setDetail(r.name)}>状态历史</a>
      ),
    },
  ]

  return (
    <Card
      title={`服务监控（共 ${data?.length ?? 0} 个，运行 ${activeCount} 个${failedCount ? `，异常 ${failedCount} 个` : ''}）`}
      extra={
        <Space>
          <Tag.CheckableTag checked={state === ''} onChange={() => setState('')}>
            全部
          </Tag.CheckableTag>
          <Tag.CheckableTag checked={state === 'active'} onChange={() => setState('active')}>
            运行中
          </Tag.CheckableTag>
          <Tag.CheckableTag checked={state === 'inactive'} onChange={() => setState('inactive')}>
            已停止
          </Tag.CheckableTag>
          <Tag.CheckableTag checked={state === 'failed'} onChange={() => setState('failed')}>
            异常
          </Tag.CheckableTag>
        </Space>
      }
    >
      <Table<ServiceState>
        rowKey="id"
        loading={isLoading}
        dataSource={data ?? []}
        columns={columns}
        pagination={false}
        size="small"
        scroll={{ y: 560 }}
      />

      <Drawer title={`服务状态历史：${detail ?? ''}`} open={!!detail} onClose={() => setDetail(null)} width={480}>
        <ServiceHistoryTable name={detail} />
      </Drawer>
    </Card>
  )
}

// ServiceHistoryTable 展示指定服务的状态变化历史。
function ServiceHistoryTable({ name }: { name: string | null }) {
  const { data } = useSWR(name ? ['/services/history', name] : null, () =>
    serviceHistory(name!, Date.now() - 7 * 24 * 3600_000, Date.now()),
  )
  const rows = (data ?? []) as ServiceState[]
  return (
    <Table<ServiceState>
      rowKey="id"
      dataSource={rows.slice().reverse().slice(0, 50)}
      pagination={false}
      size="small"
      columns={[
        { title: '时间', dataIndex: 'ts', key: 'ts', width: 160, render: (v: number) => formatTime(v) },
        {
          title: '状态',
          dataIndex: 'active_state',
          key: 'active_state',
          render: (v: string) => <Badge status={ACTIVE_COLOR[v] ?? 'default'} text={v} />,
        },
      ]}
    />
  )
}
