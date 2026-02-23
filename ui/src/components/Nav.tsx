import { NavLink } from 'react-router-dom'

export function Nav() {
  return (
    <nav className="main-nav">
      <div className="nav-brand">MyFeed</div>
      <div className="nav-links">
        <NavLink to="/" className={({ isActive }) => isActive ? 'active' : ''}>
          Feed
        </NavLink>
        <NavLink to="/peers" className={({ isActive }) => isActive ? 'active' : ''}>
          Peers
        </NavLink>
        <NavLink to="/profile" className={({ isActive }) => isActive ? 'active' : ''}>
          Profile
        </NavLink>
      </div>
    </nav>
  )
}
