import { Empty, Card } from 'antd'

// Placeholder 占位页：尚未实现的菜单功能。
export default function Placeholder() {
  return (
    <Card>
      <Empty description="该功能正在开发中，敬请期待" style={{ padding: '80px 0' }} />
    </Card>
  )
}
