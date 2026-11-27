import { Link } from '../router'
import {
  categoryIcon,
  conditionLabel,
  formatRupiah,
  listingPhotoUrl,
} from '../lib/format'
import { Icon } from './ui'

export function ListingCard({ listing, showStatus = false }) {
  const photo = listingPhotoUrl(listing)
  const inactive = listing.is_active === false || listing.takedown_at

  return (
    <Link to={`/barang/${listing.id}`} className="listing-card">
      <div className="listing-thumb">
        {photo ? (
          <img src={photo} alt={listing.title} loading="lazy" />
        ) : (
          <div className="thumb-placeholder">
            <Icon name={categoryIcon(listing.category?.slug)} size={44} />
          </div>
        )}
        <div className="thumb-top">
          {showStatus ? (
            <span className={`badge ${inactive ? 'badge-danger' : 'badge-success'}`}>
              {inactive ? 'Tidak aktif' : 'Aktif'}
            </span>
          ) : (
            <span className="badge badge-success">
              <Icon name="check_circle" filled size={13} /> Tersedia
            </span>
          )}
          <span className="badge badge-yellow">Kondisi {conditionLabel(listing.condition_percent)}</span>
        </div>
        <div className="thumb-bottom">
          <span className="badge badge-dark">
            <Icon name="place" size={13} /> {listing.campus_location?.name || 'FT UNESA'}
          </span>
        </div>
      </div>
      <div className="listing-body">
        <span className="listing-meta">
          <Icon name={categoryIcon(listing.category?.slug)} size={14} />
          {listing.category?.name || 'Lainnya'}
        </span>
        <div className="listing-title clamp-2">{listing.title}</div>
        <div className="listing-foot">
          <div className="price">
            {formatRupiah(listing.price_per_day)} <small>/ hari</small>
          </div>
          <span className="listing-meta" title="Penyedia terverifikasi KTM">
            <Icon name="verified" filled size={15} />
            KTM
          </span>
        </div>
      </div>
    </Link>
  )
}

export function ListingCardSkeleton() {
  return (
    <div className="listing-card" aria-hidden="true">
      <div className="skeleton" style={{ aspectRatio: '4 / 3', borderRadius: 0 }} />
      <div className="listing-body">
        <div className="skeleton" style={{ height: 10, width: '45%' }} />
        <div className="skeleton" style={{ height: 15, width: '90%' }} />
        <div className="skeleton" style={{ height: 15, width: '65%' }} />
        <div className="listing-foot">
          <div className="skeleton" style={{ height: 16, width: 90 }} />
          <div className="skeleton" style={{ height: 14, width: 40 }} />
        </div>
      </div>
    </div>
  )
}
