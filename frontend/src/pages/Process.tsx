import { useState } from 'react'
import { Button, Card, Drawer, Space, Table, Tag } from 'antd'
import useSWR from 'swr'
import { listProcesses, processHistory } from '../api/process'
import LineChart from '../components/charts/LineChart'
import type { ProcessSample } from '../types'
import { formatBytes, formatTime } from '../utils/format'

const STATE_COLORS: Record<string, string> = {
  R: 'green',
  S: 'blue',
  D: 'red',
  Z: 'red',
  T: 'orange',
}

// Process 进程监控页：top N 进程 + 单个进程历史曲线。
export default function Process() {
  const [sort, setSort] = useState<'cpu' | 'mem'>('cpu')
  const [detail, setDetail] = useState<{ name: string; pid: number } | null>(null)

  const { data, isLoading } = useSWR(['/process', sort], () => listProcesses(20, sort), {
    refreshInterval: 10000,
  })
  const { data: his } = useSWR(
    detail ? ['/process/history', detail.name] : null,
    () => processHistory(detail!.name, Date.now() - 24 * 3600_000, Date.now()),
  )

  const columns = [
    { title: 'PID', dataIndex: 'pid', key: 'pid', width: 80 },
    { title: '进程名', dataIndex: 'name', key: 'name', width: 180, render: (v: string) => v || '—' },
    { title: '用户', dataIndex: 'user', key: 'user', width: 110, render: (v: string) => v || '—' },
    {
      title: 'CPU %',
      dataIndex: 'cpu_percent',
      key: 'cpu_percent',
      width: 90,
      sorter: (a: ProcessSample, b: ProcessSample) => a.cpu_percent - b.cpu_percent,
      defaultSortOrder: 'descend' as const,
      render: (v: number) => <span style={{ color: v > 80 ? '#fa541c' : undefined }}>{v.toFixed(1)}%</span>,
    },
    {
      title: '内存 %',
      dataIndex: 'mem_percent',
      key: 'mem_percent',
      width: 90,
      render: (v: number) => v.toFixed(1) + '%',
    },
    { title: '内存 RSS', dataIndex: 'mem_rss', key: 'mem_rss', width: 110, render: (v: number) => formatBytes(v) },
    {
      title: '状态',
      dataIndex: 'state',
      key: 'state',
      width: 70,
      render: (v: string) => (v ? <Tag color={STATE_COLORS[v] ?? 'default'}>{v}</Tag> : '—'),
    },
    { title: '命令行', dataIndex: 'cmdline', key: 'cmdline', ellipsis: true, render: (v: string) => v || '—' },
    {
      title: '操作',
      key: 'action',
      width: 90,
      render: (_: any, r: ProcessSample) => (
        <Button size="small" onClick={() => setDetail({ name: r.name || `pid-${r.pid}`, pid: r.pid })}>
          历史
        </Button>
      ),
    },
  ]

  return (
    <Card
      title={`进程监控（按 ${sort === 'cpu' ? 'CPU' : '内存'} 排序，每 10s 刷新）`}
      extra={
        <Space>
          <Button size="small" type={sort === 'cpu' ? 'primary' : 'default'} onClick={() => setSort('cpu')}>
            按 CPU
          </Button>
          <Button size="small" type={sort === 'mem' ? 'primary' : 'default'} onClick={() => setSort('mem')}>
            按内存
          </Button>
        </Space>
      }
    >
      <Table<ProcessSample>
        rowKey="id"
        loading={isLoading}
        dataSource={data ?? []}
        columns={columns}
        pagination={false}
        size="small"
        scroll={{ y: 560 }}
      />

      <Drawer
        title={`进程历史：${detail?.name ?? ''}（近 24 小时）`}
        open={!!detail}
        onClose={() => setDetail(null)}
        width={560}
      >
        {detail && (
          <>
            <div style={{ color: '#999', marginBottom: 8 }}>PID: {detail.pid} ｜ {formatTime(Date.now())}</div>
            <LineChart
              series={[
                { name: 'CPU %', points: his?.cpu ?? [], color: '#fa541c' },
                { name: '内存 %', points: his?.mem ?? [], color: '#1677ff' },
              ]}
              formatter={(v) => v.toFixed(1) + '%'}
              height={320}
            />
          </>
        )}
      </Drawer>
    </Card>
  )
}
