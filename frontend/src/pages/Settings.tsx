import { useState } from 'react'
import { App, Button, Card, Col, Form, Input, InputNumber, Row, Space, Switch, Typography } from 'antd'
import { SendOutlined } from '@ant-design/icons'
import useSWR from 'swr'
import { getSettings, testSMTPSetting, updateSettings, type SMTPSetting } from '../api/setting'

// Settings 系统设置页：采集间隔、数据保留策略与邮件 SMTP 告警配置。
export default function Settings() {
  const [smtpTesting, setSmtpTesting] = useState(false)
  const { message } = App.useApp()
  const { data, isLoading, mutate } = useSWR('/settings', getSettings)
  const [form] = Form.useForm()
  const [smtpForm] = Form.useForm()

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

  const onSMTPTest = async () => {
    const values = await smtpForm.validateFields()
    const smtp: SMTPSetting = {
      ...values,
      to: values.to ? (values.to as string).split(',').map((s: string) => s.trim()).filter(Boolean) : [],
    }
    setSmtpTesting(true)
    try {
      await testSMTPSetting(smtp)
      message.success('测试邮件发送成功，请检查收件箱')
    } catch (e: any) {
      message.error(e.message || '测试发送失败')
    } finally {
      setSmtpTesting(false)
    }
  }

  const onSMTPSubmit = async () => {
    const values = await smtpForm.validateFields()
    const smtp: SMTPSetting = {
      ...values,
      to: values.to ? (values.to as string).split(',').map((s: string) => s.trim()).filter(Boolean) : [],
    }
    try {
      await updateSettings({ smtp })
      message.success('邮件 SMTP 配置已保存，可在「告警规则」中绑定「邮件告警」渠道')
      mutate()
    } catch (e: any) {
      message.error(e.message || '保存失败')
    }
  }

  if (isLoading || !data) {
    return <Card loading />
  }

  const smtp: SMTPSetting = data.smtp ?? { host: '', port: 465, user: '', password: '', from: '', to: [], tls: true, insecure_skip_verify: false, enabled: true }

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

        <Row gutter={16} style={{ marginTop: 16 }}>
          <Col xs={24}>
            <Card
              title="邮件 SMTP（邮件告警）"
              extra={
                <Space>
                  <Button icon={<SendOutlined />} loading={smtpTesting} onClick={onSMTPTest}>
                    测试发送
                  </Button>
                  <Button type="primary" onClick={onSMTPSubmit}>
                    保存 SMTP
                  </Button>
                </Space>
              }
            >
              <Typography.Paragraph type="secondary" style={{ fontSize: 12, marginTop: 0 }}>
                配置后自动生成「邮件告警」通知渠道，在「告警规则」中绑定该渠道即可通过邮件接收告警。密码留空表示不修改。
              </Typography.Paragraph>
              <Form
                form={smtpForm}
                layout="vertical"
                onFinish={onSMTPSubmit}
                initialValues={{
                  ...smtp,
                  password: smtp.password === '***' ? '' : smtp.password,
                  to: (smtp.to ?? []).join(','),
                }}
              >
                <Row gutter={16}>
                  <Col xs={24} md={8}>
                    <Form.Item name="host" label="SMTP 服务器" rules={[{ required: true, message: '请输入服务器' }]}>
                      <Input placeholder="如 smtp.qq.com" />
                    </Form.Item>
                  </Col>
                  <Col xs={12} md={4}>
                    <Form.Item name="port" label="端口" rules={[{ required: true }]}>
                      <InputNumber min={1} max={65535} style={{ width: '100%' }} />
                    </Form.Item>
                  </Col>
                  <Col xs={12} md={4}>
                    <Form.Item name="tls" label="SSL 直连（465）" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                  </Col>
                  <Col xs={12} md={8}>
                    <Form.Item name="enabled" label="启用邮件告警" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="user" label="邮箱账号" rules={[{ required: true, message: '请输入邮箱账号' }]}>
                      <Input placeholder="发件邮箱账号" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="password" label="授权码/密码" extra="留空=不修改">
                      <Input.Password placeholder="首次配置必填" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="from" label="发件人邮箱" rules={[{ required: true, message: '请输入发件人' }]}>
                      <Input placeholder="发件人邮箱" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name="to" label="收件人（多个用英文逗号分隔）" rules={[{ required: true }]}>
                      <Input placeholder="a@x.com,b@x.com" />
                    </Form.Item>
                  </Col>
                  <Col xs={24}>
                    <Form.Item name="insecure_skip_verify" label="忽略证书校验（自签名证书时开启）" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                  </Col>
                </Row>
              </Form>
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
