import { App, Button, Card, Col, Form, InputNumber, Row, Typography } from 'antd'
import useSWR from 'swr'
import { getSettings, updateSettings } from '../api/setting'

// Settings 系统设置页：采集间隔与数据保留策略。
export default function Settings() {
  const { message } = App.useApp()
  const { data, isLoading, mutate } = useSWR('/settings', getSettings)
  const [form] = Form.useForm()

  const onSave = async () => {
    const values = await form.validateFields()
    try {
      await updateSettings(values)
      message.success('设置已保存（采集间隔改动将在下一轮采集生效）')
      mutate()
    } catch (e: any) {
      message.error(e.message || '保存失败')
    }
  }

  if (isLoading || !data) {
    return <Card loading />
  }

  return (
    <div>
      <Form
        form={form}
        layout="vertical"
        initialValues={{
          collect_interval_sec: data.collect_interval_sec,
          process_interval_sec: data.process_interval_sec,
          service_interval_sec: data.service_interval_sec,
          process_top_n: data.process_top_n,
          snapshot_retain_days: data.snapshot_retain_days,
          process_retain_days: data.process_retain_days,
          service_retain_days: data.service_retain_days,
          alert_retain_days: data.alert_retain_days,
          notify_log_retain_days: data.notify_log_retain_days,
        }}
      >
        <Row gutter={16}>
          <Col xs={24} lg={12}>
            <Card title="采集参数">
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="collect_interval_sec" label="主指标采集周期（秒）" rules={[{ required: true }]}>
                    <InputNumber min={2} max={3600} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="process_interval_sec" label="进程采样周期（秒）" rules={[{ required: true }]}>
                    <InputNumber min={5} max={3600} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="service_interval_sec" label="服务状态采集周期（秒）" rules={[{ required: true }]}>
                    <InputNumber min={5} max={3600} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="process_top_n" label="保留进程 Top N" rules={[{ required: true }]}>
                    <InputNumber min={5} max={100} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card title="数据保留策略">
              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="snapshot_retain_days" label="指标快照保留（天）" rules={[{ required: true }]}>
                    <InputNumber min={1} max={365} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="process_retain_days" label="进程采样保留（天）" rules={[{ required: true }]}>
                    <InputNumber min={1} max={90} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="service_retain_days" label="服务状态保留（天）" rules={[{ required: true }]}>
                    <InputNumber min={1} max={90} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="alert_retain_days" label="告警事件保留（天）" rules={[{ required: true }]}>
                    <InputNumber min={1} max={365} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="notify_log_retain_days" label="通知日志保留（天）" rules={[{ required: true }]}>
                    <InputNumber min={1} max={365} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>
              <Typography.Paragraph type="secondary" style={{ marginTop: 8, fontSize: 12 }}>
                清理任务每小时自动运行（启动时也会执行一次），按保留天数分批删除过期数据。
              </Typography.Paragraph>
            </Card>
          </Col>
        </Row>
        <div style={{ marginTop: 16 }}>
          <Button type="primary" onClick={onSave}>
            保存设置
          </Button>
        </div>
      </Form>
    </div>
  )
}
