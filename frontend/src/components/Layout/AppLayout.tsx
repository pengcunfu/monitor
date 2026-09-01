import { useMemo, useState } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Avatar, Dropdown, Layout, Menu, Modal, Form, Input, App } from 'antd'
import {
  AlertOutlined,
  AppstoreOutlined,
  BellOutlined,
  ControlOutlined,
  DashboardOutlined,
  DeploymentUnitOutlined,
  FundOutlined,
  HistoryOutlined,
  LogoutOutlined,
  NotificationOutlined,
  SettingOutlined,
  ThunderboltFilled,
  UserOutlined,
} from '@ant-design/icons'
import { changePassword } from '../../api/auth'
import { useAuthStore } from '../../store/auth'

const { Sider, Header, Content } = Layout

const MENU_ITEMS = [
  { key: '/overview', icon: <DashboardOutlined />, label: '总览' },
  { key: '/realtime', icon: <FundOutlined />, label: '实时大屏' },
  { key: '/history', icon: <HistoryOutlined />, label: '历史查询' },
  { key: '/process', icon: <DeploymentUnitOutlined />, label: '进程监控' },
  { key: '/service', icon: <AppstoreOutlined />, label: '服务监控' },
  { key: '/alerts', icon: <AlertOutlined />, label: '告警中心' },
  { key: '/rules', icon: <ControlOutlined />, label: '告警规则' },
  { key: '/channels', icon: <NotificationOutlined />, label: '通知渠道' },
  { key: '/settings', icon: <SettingOutlined />, label: '系统设置' },
]

export default function AppLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const [pwdOpen, setPwdOpen] = useState(false)
  const [pwdForm] = Form.useForm()
  const navigate = useNavigate()
  const location = useLocation()
  const { user, logout } = useAuthStore()
  const { message } = App.useApp()

  const selectedKey = useMemo(() => {
    const path = location.pathname
    const match = MENU_ITEMS.find((m) => path.startsWith(m.key))
    return match ? match.key : '/overview'
  }, [location.pathname])

  const onLogout = () => {
    logout()
    navigate('/login')
  }

  const onChangepwd = async () => {
    const values = await pwdForm.validateFields()
    try {
      await changePassword(values.old_password, values.new_password)
      message.success('密码修改成功')
      setPwdOpen(false)
      pwdForm.resetFields()
    } catch (e: any) {
      message.error(e.message || '修改失败')
    }
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider collapsible collapsed={collapsed} onCollapse={setCollapsed} theme="dark">
        <div
          style={{
            minHeight: 48,
            margin: 12,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            textAlign: 'center',
            color: '#fff',
            fontSize: collapsed ? 14 : 13,
            fontWeight: 600,
            lineHeight: 1.4,
          }}
        >
          {collapsed ? (
            <ThunderboltFilled style={{ fontSize: 20 }} />
          ) : (
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
              <ThunderboltFilled style={{ fontSize: 16 }} />
              熔岩网络安全事件应急处置系统
            </span>
          )}
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          items={MENU_ITEMS}
          onClick={({ key }) => navigate(key)}
        />
      </Sider>
      <Layout>
        <Header
          style={{
            background: '#fff',
            padding: '0 16px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          <div style={{ fontSize: 16, fontWeight: 600 }}>
            {MENU_ITEMS.find((m) => m.key === selectedKey)?.label ?? '总览'}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <BellOutlined style={{ fontSize: 18, color: '#666', cursor: 'pointer' }} />
            <Dropdown
              menu={{
                items: [
                  { key: 'pwd', icon: <SettingOutlined />, label: '修改密码' },
                  { type: 'divider' },
                  { key: 'logout', icon: <LogoutOutlined />, label: '退出登录', danger: true },
                ],
                onClick: ({ key }) => {
                  if (key === 'logout') onLogout()
                  if (key === 'pwd') setPwdOpen(true)
                },
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
                <Avatar size="small" icon={<UserOutlined />} style={{ background: '#1677ff' }} />
                <span>{user?.username ?? 'admin'}</span>
              </div>
            </Dropdown>
          </div>
        </Header>
        <Content style={{ margin: 16 }}>
          <Outlet />
        </Content>
      </Layout>

      <Modal title="修改密码" open={pwdOpen} onOk={onChangepwd} onCancel={() => setPwdOpen(false)} destroyOnHidden>
        <Form form={pwdForm} layout="vertical">
          <Form.Item name="old_password" label="原密码" rules={[{ required: true, message: '请输入原密码' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="new_password"
            label="新密码"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '至少 6 位' },
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  )
}
