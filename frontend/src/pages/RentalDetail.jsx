import { useCallback, useEffect, useRef, useState } from 'react'
import { Link, useLocation } from '../router'
import { useAuth } from '../auth/useAuth'
import { rentalApi } from '../lib/api'
import { useApi } from '../lib/useApi'
import { saveBlobUrl, wsUrl } from '../lib/http'
import {
  dateRange,
  fileSize,
  formatDate,
  formatDateTime,
  formatRupiah,
  rentalStatus,
  timeAgo,
  todayISO,
} from '../lib/format'
import {
  Alert,
  Avatar,
  ErrorBox,
  Icon,
  Loading,
  Modal,
} from '../components/ui'

function ChatPanel({ rentalId, canChat, user }) {
  const [messages, setMessages] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const [connected, setConnected] = useState(false)
  const [note, setNote] = useState(null)

  const scrollRef = useRef(null)
  const socketRef = useRef(null)
  const retryRef = useRef(null)
  const attemptsRef = useRef(0)

  const append = useCallback((message) => {
    if (!message?.id) return
    setMessages((prev) => (prev.some((item) => item.id === message.id) ? prev : [...prev, message]))
  }, [])

  const load = useCallback(async () => {
    try {
      const data = await rentalApi.messages(rentalId, { limit: 100 })
      setMessages(data.messages || [])
      setError(null)
    } catch (err) {
      setError(err)
    } finally {
      setLoading(false)
    }
  }, [rentalId])

  useEffect(() => {
    setMessages([])
    setLoading(true)
    setError(null)
    load()
  }, [load])

  useEffect(() => {
    if (!canChat) {
      setConnected(false)
      return undefined
    }

    let disposed = false
    attemptsRef.current = 0

    const connect = async () => {
      try {
        const ticket = await rentalApi.wsTicket(rentalId)
        if (disposed) return

        const socket = new WebSocket(wsUrl(ticket.ws_path))
        socketRef.current = socket

        socket.onopen = () => {
          if (disposed) return
          attemptsRef.current = 0
          setConnected(true)
        }

        socket.onmessage = (event) => {
          try {
            const payload = JSON.parse(event.data)
            if (payload.type === 'message' && payload.message) {
              append(payload.message)
            } else if (payload.type === 'error') {
              setNote(payload.message || payload.code || 'Pesan gagal dikirim.')
            }
          } catch {
            // frame tidak dikenal — abaikan
          }
        }

        socket.onclose = () => {
          if (disposed) return
          setConnected(false)
          if (attemptsRef.current >= 3) return
          attemptsRef.current += 1
          retryRef.current = setTimeout(connect, 2000 * attemptsRef.current)
        }

        socket.onerror = () => {
          try {
            socket.close()
          } catch {
            // abaikan
          }
        }
      } catch {
        if (!disposed) setConnected(false)
      }
    }

    connect()

    // Cadangan bila WebSocket diblokir jaringan kampus.
    const poll = setInterval(load, 15000)

    return () => {
      disposed = true
      clearInterval(poll)
      if (retryRef.current) clearTimeout(retryRef.current)
      const socket = socketRef.current
      socketRef.current = null
      if (socket && [WebSocket.OPEN, WebSocket.CONNECTING].includes(socket.readyState)) {
        try {
          socket.close()
        } catch {
          // abaikan
        }
      }
      setConnected(false)
    }
  }, [rentalId, canChat, append, load])

  useEffect(() => {
    const node = scrollRef.current
    if (node) node.scrollTop = node.scrollHeight
  }, [messages.length])

  const submit = async (event) => {
    event.preventDefault()
    const body = draft.trim()
    if (!body || sending) return

    setSending(true)
    setNote(null)
    try {
      const data = await rentalApi.sendMessage(rentalId, body)
      append(data.message)
      setDraft('')
    } catch (err) {
      setNote(err.message)
    } finally {
      setSending(false)
    }
  }

  return (
    <div className="chat">
      <div className="between">
        <span className="tiny muted">
          {messages.length} pesan · {canChat ? 'percakapan aktif' : 'percakapan ditutup'}
        </span>
        <span className={`chat-status ${connected ? '' : 'offline'}`.trim()}>
          <span className="dot" />
          {connected ? 'Realtime aktif' : canChat ? 'Sinkron berkala' : 'Hanya baca'}
        </span>
      </div>

      <div className="chat-scroll" ref={scrollRef}>
        {loading ? (
          <div className="chat-empty">Memuat pesan…</div>
        ) : error ? (
          <div className="chat-empty">
            Gagal memuat pesan. <button type="button" className="link-btn" onClick={load}>Coba lagi</button>
          </div>
        ) : messages.length ? (
          messages.map((message) => {
            const mine = message.sender?.id === user?.id
            return (
              <div key={message.id} className={`bubble-row ${mine ? 'mine' : ''}`.trim()}>
                {!mine ? <Avatar name={message.sender?.name} size="sm" /> : null}
                <div className={`bubble ${mine ? 'mine' : ''}`.trim()}>
                  {!mine ? <div className="tiny strong" style={{ marginBottom: 2 }}>{message.sender?.name}</div> : null}
                  {message.body}
                  <div className="bubble-meta">{formatDate(message.created_at, 'datetime')}</div>
                </div>
              </div>
            )
          })
        ) : (
          <div className="chat-empty">
            Belum ada pesan. Sapa pihak lain untuk menyepakati waktu dan titik COD.
          </div>
        )}
      </div>

      {note ? <Alert tone="error" icon="error">{note}</Alert> : null}

      {canChat ? (
        <form className="chat-form" onSubmit={submit}>
          <input
            className="input"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="Tulis pesan untuk pihak lain…"
            maxLength={1000}
            aria-label="Isi pesan"
          />
          <button type="submit" className="btn btn-primary" disabled={sending || !draft.trim()}>
            {sending ? <span className="spinner" /> : <Icon name="send" />}
            Kirim
          </button>
        </form>
      ) : (
        <Alert tone="info" icon="info">
          Chat ditutup karena transaksi sudah tidak aktif. Riwayat pesan tetap dapat dibaca.
        </Alert>
      )}
    </div>
  )
}

const DOCUMENT_LABELS = {
  agreement: 'Surat Perjanjian',
  handover_proof: 'Bukti Serah Terima',
  return_proof: 'Bukti Pengembalian',
  cod_receipt: 'Kuitansi COD',
  other: 'Dokumen Tambahan',
}

function documentFilename(type) {
  switch (type) {
    case 'agreement': return 'surat-perjanjian-sewa.pdf'
    case 'cod_receipt': return 'kuitansi-cod.pdf'
    case 'handover_proof': return 'bukti-serah-terima.jpg'
    case 'return_proof': return 'bukti-pengembalian.jpg'
    default: return 'dokumen-rental.pdf'
  }
}

function validateProof(file) {
  if (!file) throw new Error('Pilih foto bukti terlebih dahulu.')
  if (!['image/jpeg', 'image/png'].includes(file.type)) {
    throw new Error('Bukti harus berupa foto JPG atau PNG.')
  }
  if (file.size > 10 * 1024 * 1024) throw new Error('Ukuran bukti maksimal 10 MB.')
}

function ProofForm({ title, hint, note, onNote, file, onFile, busy, onSubmit, buttonLabel }) {
  return (
    <div className="stack-sm">
      <div className="strong">{title}</div>
      <p className="hint mt-1">{hint}</p>
      <div className="field">
        <label className="label">Foto bukti <span className="req">*</span></label>
        <input
          className="input"
          type="file"
          accept="image/jpeg,image/png"
          onChange={(event) => onFile(event.target.files?.[0] || null)}
          aria-label={title}
        />
      </div>
      <div className="field">
        <label className="label">Catatan <span className="muted">(opsional)</span></label>
        <textarea
          className="textarea"
          rows={2}
          value={note}
          onChange={(event) => onNote(event.target.value)}
          placeholder="Tuliskan kondisi barang atau titik temu."
        />
      </div>
      <div className="row-wrap">
        <button type="button" className="btn btn-primary btn-sm" disabled={busy || !file} onClick={onSubmit}>
          {busy ? <span className="spinner" /> : <Icon name="upload" />}
          {buttonLabel}
        </button>
        <span className="tiny muted">JPG atau PNG · maksimal 10 MB</span>
      </div>
    </div>
  )
}

function CODPanel({ rentalId, rental, cod, onChanged }) {
  const { data, loading, error, reload } = useApi(() => rentalApi.documents(rentalId), [rentalId])
  const [busy, setBusy] = useState(null)
  const [actionError, setActionError] = useState(null)
  const [feedback, setFeedback] = useState(null)
  const [handoverFile, setHandoverFile] = useState(null)
  const [handoverNote, setHandoverNote] = useState('')
  const [returnFile, setReturnFile] = useState(null)
  const [returnNote, setReturnNote] = useState('')
  const [settleOpen, setSettleOpen] = useState(false)
  const [amountPaid, setAmountPaid] = useState(String(rental?.total_price ?? 0))
  const [paymentMethod, setPaymentMethod] = useState('cash')
  const [settleNotes, setSettleNotes] = useState('')
  const [settleError, setSettleError] = useState(null)

  const isOwner = rental?.my_role === 'owner'
  const isRenter = rental?.my_role === 'renter'
  const isActive = rental?.status === 'approved'
  const codComplete = Boolean(cod?.handover_confirmed_at && cod?.return_confirmed_at)

  const steps = [
    { label: 'Disetujui', at: cod?.agreed_at },
    { label: 'Barang diserahkan', at: cod?.handover_at },
    { label: 'Dikonfirmasi penyewa', at: cod?.handover_confirmed_at },
    { label: 'Barang dikembalikan', at: cod?.return_at },
    { label: 'Dikonfirmasi penyedia', at: cod?.return_confirmed_at },
    { label: 'Pembayaran dicatat', at: cod?.settled_at },
  ]

  const run = async (key, action, message) => {
    setBusy(key)
    setActionError(null)
    setFeedback(null)
    try {
      await action()
      await reload()
      await onChanged()
      setFeedback({ tone: 'success', message })
    } catch (err) {
      setActionError(err)
    } finally {
      setBusy(null)
    }
  }

  const submitHandover = () => {
    try {
      validateProof(handoverFile)
    } catch (err) {
      setActionError(err)
      return
    }
    run(
      'handover',
      () => rentalApi.handover(rentalId, handoverFile, handoverNote.trim()),
      'Bukti serah terima diunggah. Penyewa akan diminta mengonfirmasi.',
    ).then(() => {
      setHandoverFile(null)
      setHandoverNote('')
    })
  }

  const submitReturn = () => {
    try {
      validateProof(returnFile)
    } catch (err) {
      setActionError(err)
      return
    }
    run(
      'return',
      () => rentalApi.returnProof(rentalId, returnFile, returnNote.trim()),
      'Bukti pengembalian diunggah. Penyedia akan diminta mengonfirmasi.',
    ).then(() => {
      setReturnFile(null)
      setReturnNote('')
    })
  }

  const submitSettle = async (event) => {
    event.preventDefault()
    const amount = Number(amountPaid)
    if (!Number.isFinite(amount) || amount < 0) {
      setSettleError({ message: 'Jumlah pembayaran harus berupa angka.' })
      return
    }
    setBusy('settle')
    setSettleError(null)
    try {
      await rentalApi.settle(rentalId, {
        amount_paid: Math.round(amount),
        payment_method: paymentMethod,
        notes: settleNotes.trim(),
      })
      await reload()
      await onChanged()
      setSettleOpen(false)
      setFeedback({ tone: 'success', message: 'Pembayaran dicatat. Kuitansi COD tersedia untuk diunduh.' })
    } catch (err) {
      setSettleError(err)
    } finally {
      setBusy(null)
    }
  }

  const downloadDocument = async (doc) => {
    setBusy(`doc-${doc.id}`)
    setActionError(null)
    try {
      const url = await rentalApi.documentBlobUrl(rentalId, doc.id)
      saveBlobUrl(url, documentFilename(doc.type))
    } catch (err) {
      setActionError(err)
    } finally {
      setBusy(null)
    }
  }

  const documents = data?.documents || []

  return (
    <section className="card card-pad-lg">
      <div className="between" style={{ marginBottom: 12 }}>
        <div>
          <h2 className="section-title" style={{ fontSize: 18 }}>COD & Dokumen</h2>
          <p className="hint mt-1">
            Serah terima, pengembalian, dan pembayaran dicatat di sini. Semua dokumen bersifat privat.
          </p>
        </div>
        <span className={`badge ${codComplete ? 'badge-success' : 'badge-muted'}`}>
          <Icon name={codComplete ? 'task_alt' : 'hourglass_top'} size={13} />
          {codComplete ? 'COD lengkap' : 'COD berjalan'}
        </span>
      </div>

      {feedback ? <Alert tone="success" icon="check_circle">{feedback.message}</Alert> : null}
      {actionError ? <ErrorBox error={actionError} title="Aksi COD gagal" /> : null}

      <h3 className="card-title mt-4">Progres serah terima</h3>
      {cod ? (
        <div className="timeline mt-3">
          {steps.map((step, index) => {
            const done = Boolean(step.at)
            const current = !done && steps.slice(0, index).every((item) => Boolean(item.at))
            return (
              <div
                key={step.label}
                className={`timeline-item ${done ? 'done' : current ? 'current' : ''}`.trim()}
              >
                <span className="timeline-dot" />
                <div className="timeline-title">{step.label}</div>
                <div className="timeline-meta">
                  {done ? formatDateTime(step.at) : current ? 'Menunggu tindakan' : 'Belum tersedia'}
                </div>
              </div>
            )
          })}
        </div>
      ) : (
        <p className="hint mt-3">Progres COD tersedia setelah pengajuan disetujui.</p>
      )}

      {isOwner && isActive && !cod?.handover_at ? (
        <div className="mt-4">
          <ProofForm
            title="Unggah bukti serah terima"
            hint="Ambil foto barang saat diserahkan kepada penyewa."
            note={handoverNote}
            onNote={setHandoverNote}
            file={handoverFile}
            onFile={setHandoverFile}
            busy={busy === 'handover'}
            onSubmit={submitHandover}
            buttonLabel="Kirim Bukti Serah Terima"
          />
        </div>
      ) : null}

      {isRenter && isActive && cod?.handover_at && !cod?.handover_confirmed_at ? (
        <div className="row-wrap mt-4">
          <button
            type="button"
            className="btn btn-primary btn-sm"
            disabled={Boolean(busy)}
            onClick={() => run(
              'confirm-handover',
              () => rentalApi.confirmHandover(rentalId),
              'Serah terima dikonfirmasi.',
            )}
          >
            {busy === 'confirm-handover' ? <span className="spinner" /> : <Icon name="check_circle" />}
            Konfirmasi Barang Diterima
          </button>
        </div>
      ) : null}

      {isRenter && isActive && cod?.handover_confirmed_at && !cod?.return_at ? (
        <div className="mt-4">
          <ProofForm
            title="Unggah bukti pengembalian"
            hint="Ambil foto barang saat dikembalikan kepada penyedia."
            note={returnNote}
            onNote={setReturnNote}
            file={returnFile}
            onFile={setReturnFile}
            busy={busy === 'return'}
            onSubmit={submitReturn}
            buttonLabel="Kirim Bukti Pengembalian"
          />
        </div>
      ) : null}

      {isOwner && isActive && cod?.return_at && !cod?.return_confirmed_at ? (
        <div className="row-wrap mt-4">
          <button
            type="button"
            className="btn btn-primary btn-sm"
            disabled={Boolean(busy)}
            onClick={() => run(
              'confirm-return',
              () => rentalApi.confirmReturn(rentalId),
              'Pengembalian dikonfirmasi.',
            )}
          >
            {busy === 'confirm-return' ? <span className="spinner" /> : <Icon name="check_circle" />}
            Konfirmasi Barang Kembali
          </button>
        </div>
      ) : null}

      {isOwner && cod?.return_confirmed_at && !cod?.settled_at ? (
        <div className="row-wrap mt-4">
          <button
            type="button"
            className="btn btn-outline btn-sm"
            onClick={() => {
              setAmountPaid(String(rental?.total_price ?? 0))
              setPaymentMethod('cash')
              setSettleNotes('')
              setSettleError(null)
              setSettleOpen(true)
            }}
          >
            <Icon name="payments" /> Catat Pembayaran
          </button>
        </div>
      ) : null}

      {isOwner && !codComplete ? (
        <Alert tone="warning" icon="hourglass_top" title="COD belum lengkap">
          Selesaikan serah terima dan pengembalian barang sebelum menandai transaksi selesai.
        </Alert>
      ) : null}

      <hr className="divider" style={{ margin: '18px 0' }} />

      <h3 className="card-title">Dokumen transaksi</h3>
      {loading ? (
        <Loading label="Memuat dokumen…" />
      ) : error ? (
        <ErrorBox error={error} onRetry={reload} title="Dokumen gagal dimuat" />
      ) : documents.length ? (
        <div className="stack-sm mt-3">
          {documents.map((doc) => (
            <div className="party" key={doc.id}>
              <span className="stat-icon"><Icon name={doc.type === 'cod_receipt' ? 'receipt_long' : 'description'} /></span>
              <div className="party-body">
                <div className="party-name">{DOCUMENT_LABELS[doc.type] || doc.type}</div>
                <div className="party-meta">
                  {formatDate(doc.created_at, 'short')} · {fileSize(doc.size_bytes)}
                </div>
              </div>
              <button
                type="button"
                className="btn btn-outline btn-sm"
                disabled={busy === `doc-${doc.id}`}
                onClick={() => downloadDocument(doc)}
              >
                {busy === `doc-${doc.id}` ? <span className="spinner" /> : <Icon name="download" />}
                Unduh
              </button>
            </div>
          ))}
        </div>
      ) : (
        <p className="hint mt-3">
          Surat perjanjian dibuat saat pengajuan disetujui. Bukti dan kuitansi muncul setelah COD
          dijalankan.
        </p>
      )}

      {settleOpen ? (
        <Modal
          title="Catat pembayaran COD"
          subtitle={`Total sewa ${formatRupiah(rental?.total_price || 0)}`}
          onClose={() => setSettleOpen(false)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setSettleOpen(false)}>Batal</button>
              <button type="submit" form="cod-settle-form" className="btn btn-primary" disabled={busy === 'settle'}>
                {busy === 'settle' ? <span className="spinner" /> : <Icon name="receipt_long" />}
                Simpan & Buat Kuitansi
              </button>
            </>
          }
        >
          <form id="cod-settle-form" className="stack" onSubmit={submitSettle}>
            <div className="field">
              <label className="label" htmlFor="cod-amount">Jumlah dibayar <span className="req">*</span></label>
              <input
                id="cod-amount"
                className="input"
                type="number"
                min="0"
                value={amountPaid}
                onChange={(event) => setAmountPaid(event.target.value)}
              />
              <span className="hint">Boleh berbeda dari total sewa bila ada kesepakatan khusus.</span>
            </div>
            <div className="field">
              <label className="label" htmlFor="cod-method">Metode pembayaran</label>
              <select
                id="cod-method"
                className="select"
                value={paymentMethod}
                onChange={(event) => setPaymentMethod(event.target.value)}
              >
                <option value="cash">Tunai (COD)</option>
                <option value="transfer">Transfer</option>
              </select>
            </div>
            <div className="field">
              <label className="label" htmlFor="cod-notes">Catatan <span className="muted">(opsional)</span></label>
              <textarea
                id="cod-notes"
                className="textarea"
                rows={2}
                value={settleNotes}
                onChange={(event) => setSettleNotes(event.target.value)}
                placeholder="Contoh: Pembayaran diterima langsung saat pengembalian barang."
              />
            </div>
            {settleError ? <Alert tone="error" icon="error">{settleError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}
    </section>
  )
}

export default function RentalDetail({ id }) {
  const { user } = useAuth()
  const { query } = useLocation()
  const { data, loading, error, reload } = useApi(() => rentalApi.detail(id), [id])
  const rental = data?.rental || null
  const codQuery = useApi(async () => {
    try {
      const result = await rentalApi.cod(id)
      return result?.cod || null
    } catch {
      return null
    }
  }, [id, rental?.status])
  const cod = codQuery.data || null

  const [busy, setBusy] = useState(null)
  const [feedback, setFeedback] = useState(null)
  const [actionError, setActionError] = useState(null)
  const [rejectOpen, setRejectOpen] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [reason, setReason] = useState('')
  const [modalBusy, setModalBusy] = useState(false)
  const [modalError, setModalError] = useState(null)
  const [agreementBusy, setAgreementBusy] = useState(false)

  const chatRef = useRef(null)

  useEffect(() => {
    if (!query.chat) return
    const timer = setTimeout(() => {
      chatRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }, 250)
    return () => clearTimeout(timer)
  }, [query.chat, rental?.id])

  if (loading) {
    return (
      <div className="container page">
        <Loading label="Memuat detail penyewaan…" />
      </div>
    )
  }

  if (error || !rental) {
    return (
      <div className="container page">
        <ErrorBox error={error} onRetry={reload} title="Penyewaan tidak bisa dimuat" />
        <div className="empty mt-4">
          <Icon name="receipt_long" size={38} />
          <strong>Transaksi tidak ditemukan</strong>
          <p>Kamu hanya bisa membuka transaksi yang melibatkan akunmu.</p>
          <Link to="/penyewaan" className="btn btn-outline btn-sm">Kembali ke Penyewaan</Link>
        </div>
      </div>
    )
  }

  const meta = rentalStatus(rental.status)
  const isOwner = rental.my_role === 'owner'
  const isRenter = rental.my_role === 'renter'
  const other = isOwner ? rental.renter : rental.owner
  const canChat = ['pending', 'approved'].includes(rental.status)
  const codComplete = Boolean(cod?.handover_confirmed_at && cod?.return_confirmed_at)
  const canComplete =
    isOwner && rental.status === 'approved' && rental.end_date <= todayISO() && codComplete
  const canCancel =
    (isOwner && rental.status === 'approved') ||
    (isRenter && ['pending', 'approved'].includes(rental.status))

  const reloadAll = async () => {
    await Promise.all([reload(), codQuery.reload()])
  }

  const refresh = async (message) => {
    await reloadAll()
    if (message) setFeedback({ tone: 'success', message })
  }

  const runAction = async (key, action, message) => {
    setBusy(key)
    setActionError(null)
    setFeedback(null)
    try {
      await action()
      await refresh(message)
    } catch (err) {
      setActionError(err)
    } finally {
      setBusy(null)
    }
  }

  const submitReasonAction = async (event) => {
    event.preventDefault()
    if (!reason.trim() && (rejectOpen || (isOwner && cancelOpen))) {
      setModalError({ message: 'Alasan wajib diisi.' })
      return
    }

    setModalBusy(true)
    setModalError(null)
    try {
      if (rejectOpen) {
        await rentalApi.reject(id, reason.trim())
        setRejectOpen(false)
        setFeedback({ tone: 'success', message: 'Pengajuan ditolak dan penyewa sudah diberitahu.' })
      } else {
        await rentalApi.cancel(id, reason.trim())
        setCancelOpen(false)
        setFeedback({ tone: 'success', message: 'Transaksi berhasil dibatalkan.' })
      }
      setReason('')
      await reload()
    } catch (err) {
      setModalError(err)
    } finally {
      setModalBusy(false)
    }
  }

  const downloadAgreement = async () => {
    setAgreementBusy(true)
    setActionError(null)
    try {
      const url = await rentalApi.agreementBlobUrl(id)
      const link = document.createElement('a')
      link.href = url
      link.download = `perjanjian-sewa-${id.slice(0, 8)}.pdf`
      document.body.appendChild(link)
      link.click()
      link.remove()
      setTimeout(() => URL.revokeObjectURL(url), 15000)
    } catch (err) {
      setActionError(err)
    } finally {
      setAgreementBusy(false)
    }
  }

  const events = rental.events || []

  return (
    <div className="container page">
      <div className="breadcrumbs">
        <Link to="/penyewaan">Penyewaan Saya</Link>
        <Icon name="chevron_right" />
        <span>#{rental.id.slice(0, 8)}</span>
      </div>

      <div className="page-head">
        <div>
          <div className="eyebrow">
            <Icon name="receipt_long" size={15} /> Detail transaksi #{rental.id.slice(0, 8).toUpperCase()}
          </div>
          <h1 className="page-title">{rental.listing?.title}</h1>
          <p className="subtitle mt-2">
            {isOwner ? 'Kamu bertindak sebagai pemilik barang.' : 'Kamu bertindak sebagai penyewa.'}{' '}
            Diajukan {timeAgo(rental.created_at)}.
          </p>
        </div>
        <div className="row-wrap">
          <span className={`badge badge-${isOwner ? 'mint' : 'primary'}`}>
            <Icon name={isOwner ? 'storefront' : 'shopping_bag'} size={13} />
            {isOwner ? 'Pemilik' : 'Penyewa'}
          </span>
          <span className={`badge badge-${meta.tone}`}>
            <Icon name={meta.icon} size={13} /> {meta.label}
          </span>
          <Link to={`/lapor?rental=${rental.id}`} className="btn btn-ghost btn-sm">
            <Icon name="flag" /> Laporkan
          </Link>
        </div>
      </div>

      {feedback ? (
        <div className="mb-4">
          <Alert tone={feedback.tone} icon="check_circle">{feedback.message}</Alert>
        </div>
      ) : null}

      {actionError ? (
        <div className="mb-4">
          <ErrorBox error={actionError} title="Aksi gagal diproses" />
        </div>
      ) : null}

      <div className="detail-grid">
        <div className="stack">
          <section className="card card-pad-lg">
            <div className="row" style={{ alignItems: 'flex-start', gap: 14, flexWrap: 'wrap' }}>
              <div
                style={{
                  width: 128,
                  height: 100,
                  borderRadius: 12,
                  overflow: 'hidden',
                  background: 'linear-gradient(135deg,#eef1ff,#fff3d1)',
                  display: 'grid',
                  placeItems: 'center',
                  color: 'var(--primary)',
                  flexShrink: 0,
                }}
              >
                {rental.listing?.cover_photo_url ? (
                  <img
                    src={rental.listing.cover_photo_url}
                    alt={rental.listing.title}
                    style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                  />
                ) : (
                  <Icon name="inventory_2" size={36} />
                )}
              </div>
              <div style={{ flex: 1, minWidth: 220 }}>
                <h2 className="section-title" style={{ fontSize: 19 }}>{rental.listing?.title}</h2>
                <div className="hint mt-1">
                  Barang milik {rental.owner?.name}. Ajukan pertanyaan lewat chat di bawah.
                </div>
                <div className="row-wrap mt-3">
                  <Link to={`/barang/${rental.listing?.id}`} className="btn btn-outline btn-sm">
                    <Icon name="visibility" /> Lihat barang
                  </Link>
                  {rental.listing?.id ? (
                    <Link to={`/pesan`} className="btn btn-ghost btn-sm">
                      <Icon name="forum" /> Semua percakapan
                    </Link>
                  ) : null}
                </div>
              </div>
            </div>

            <hr className="divider" style={{ margin: '18px 0' }} />

            <div className="grid cols-4">
              <div className="spec">
                <span>Periode sewa</span>
                <strong>{dateRange(rental.start_date, rental.end_date)}</strong>
              </div>
              <div className="spec">
                <span>Durasi</span>
                <strong>{rental.total_days} hari</strong>
              </div>
              <div className="spec">
                <span>Tarif harian</span>
                <strong>{formatRupiah(rental.price_per_day_snapshot)}</strong>
              </div>
              <div className="spec">
                <span>Total (COD)</span>
                <strong style={{ color: 'var(--primary)' }}>{formatRupiah(rental.total_price)}</strong>
              </div>
            </div>
          </section>

          <section className="card card-pad-lg">
            <h2 className="section-title" style={{ fontSize: 18 }}>Pihak transaksi</h2>
            <div className="grid cols-2 mt-3">
              <div className="party">
                <Avatar name={rental.renter?.name} tone="yellow" />
                <div className="party-body">
                  <div className="party-name">{rental.renter?.name}</div>
                  <div className="party-meta">Penyewa</div>
                </div>
                {isRenter ? <span className="badge badge-primary">Kamu</span> : null}
              </div>
              <div className="party">
                <Avatar name={rental.owner?.name} tone="mint" />
                <div className="party-body">
                  <div className="party-name">{rental.owner?.name}</div>
                  <div className="party-meta">Pemilik barang</div>
                </div>
                {isOwner ? <span className="badge badge-mint">Kamu</span> : null}
              </div>
            </div>
            <p className="hint mt-3">
              Kontak pribadi tidak ditampilkan. Gunakan chat untuk menyepakati waktu dan titik COD di
              area Fakultas Teknik.
            </p>
          </section>

          <section className="card card-pad-lg">
            <h2 className="section-title" style={{ fontSize: 18 }}>Riwayat status</h2>
            {events.length ? (
              <div className="timeline mt-4">
                {events.map((event, index) => {
                  const status = rentalStatus(event.to_status)
                  const current = index === events.length - 1
                  return (
                    <div
                      key={event.id ?? index}
                      className={`timeline-item ${current ? 'current' : 'done'}`.trim()}
                    >
                      <span className="timeline-dot" />
                      <div className="timeline-title">
                        {status.label}
                        {event.actor?.name ? <span className="muted"> · oleh {event.actor.name}</span> : null}
                      </div>
                      <div className="timeline-meta">{formatDateTime(event.created_at)}</div>
                      {event.note ? <div className="tiny body-text mt-1">“{event.note}”</div> : null}
                    </div>
                  )
                })}
              </div>
            ) : (
              <p className="hint mt-3">Belum ada riwayat perubahan status.</p>
            )}

            {rental.rejection_reason ? (
              <div className="alert alert-error mt-4">
                <Icon name="cancel" />
                <div className="alert-body">
                  <strong>Alasan penolakan</strong>
                  {rental.rejection_reason}
                </div>
              </div>
            ) : null}
          </section>

          <CODPanel rentalId={rental.id} rental={rental} cod={cod} onChanged={reloadAll} />

          <section className="card card-pad-lg" ref={chatRef} id="chat">
            <div className="between" style={{ marginBottom: 12 }}>
              <div>
                <h2 className="section-title" style={{ fontSize: 18 }}>Chat & koordinasi</h2>
                <p className="hint mt-1">
                  Sepakati jam temu, kondisi barang, dan pengembalian lewat chat.
                </p>
              </div>
              <span className="badge badge-muted">
                <Icon name={canChat ? 'chat' : 'lock'} size={13} />
                {canChat ? 'Aktif' : 'Ditutup'}
              </span>
            </div>
            <ChatPanel rentalId={rental.id} canChat={canChat} user={user} />
          </section>
        </div>

        <aside className="sticky-side stack">
          <section className="card card-pad-lg">
            <span className="eyebrow">Total pembayaran COD</span>
            <div className="price price-lg mt-1">{formatRupiah(rental.total_price)}</div>
            <p className="hint mt-1">
              {formatRupiah(rental.price_per_day_snapshot)} × {rental.total_days} hari. Dibayar tunai
              saat serah-terima.
            </p>

            <hr className="divider" style={{ margin: '16px 0' }} />

            {isOwner && rental.status === 'pending' ? (
              <div className="stack">
                <button
                  type="button"
                  className="btn btn-primary btn-block"
                  disabled={busy === 'approve'}
                  onClick={() => runAction('approve', () => rentalApi.approve(id), 'Pengajuan disetujui. Penyewa menerima notifikasi.')}
                >
                  {busy === 'approve' ? <span className="spinner" /> : <Icon name="check_circle" />}
                  Setujui Pengajuan
                </button>
                <button
                  type="button"
                  className="btn btn-danger-soft btn-block"
                  disabled={Boolean(busy)}
                  onClick={() => {
                    setReason('')
                    setModalError(null)
                    setRejectOpen(true)
                  }}
                >
                  <Icon name="cancel" /> Tolak Pengajuan
                </button>
              </div>
            ) : null}

            {isRenter && rental.status === 'pending' ? (
              <div className="stack">
                <Alert tone="warning" icon="hourglass_top" title="Menunggu persetujuan">
                  Pemilik barang akan meninjau pengajuanmu. Kamu bisa berdiskusi lewat chat.
                </Alert>
                <button
                  type="button"
                  className="btn btn-danger-soft btn-block"
                  disabled={Boolean(busy)}
                  onClick={() => {
                    setReason('')
                    setModalError(null)
                    setCancelOpen(true)
                  }}
                >
                  <Icon name="block" /> Batalkan Pengajuan
                </button>
              </div>
            ) : null}

            {rental.status === 'approved' ? (
              <div className="stack">
                <Alert tone="success" icon="check_circle" title="Sewa disetujui">
                  Selesaikan COD sesuai kesepakatan. Tandai selesai setelah barang dikembalikan.
                </Alert>

                <button
                  type="button"
                  className="btn btn-outline btn-block"
                  disabled={agreementBusy}
                  onClick={downloadAgreement}
                >
                  {agreementBusy ? <span className="spinner" /> : <Icon name="download" />}
                  Unduh Surat Perjanjian
                </button>

                {isOwner ? (
                  <button
                    type="button"
                    className="btn btn-primary btn-block"
                    disabled={!canComplete || busy === 'complete'}
                    onClick={() => runAction('complete', () => rentalApi.complete(id), 'Transaksi ditandai selesai.')}
                  >
                    {busy === 'complete' ? <span className="spinner" /> : <Icon name="task_alt" />}
                    Tandai Selesai
                  </button>
                ) : null}

                {isOwner && !codComplete ? (
                  <Alert tone="warning" icon="hourglass_top">
                    Selesaikan serah terima dan pengembalian barang di panel COD sebelum menandai selesai.
                  </Alert>
                ) : null}

                {isOwner && codComplete && !canComplete ? (
                  <p className="hint">
                    Tombol selesai aktif setelah tanggal sewa berakhir ({formatDate(rental.end_date, 'short')}).
                  </p>
                ) : null}

                <button
                  type="button"
                  className="btn btn-danger-soft btn-block"
                  disabled={Boolean(busy)}
                  onClick={() => {
                    setReason('')
                    setModalError(null)
                    setCancelOpen(true)
                  }}
                >
                  <Icon name="block" /> Batalkan Sewa
                </button>
              </div>
            ) : null}

            {['rejected', 'cancelled', 'completed'].includes(rental.status) ? (
              <div className="stack">
                <Alert tone={rental.status === 'completed' ? 'success' : 'info'} icon={meta.icon} title={meta.label}>
                  {rental.status === 'completed'
                    ? 'Transaksi selesai. Terima kasih telah menjaga ekosistem sewa FT UNESA.'
                    : rental.status === 'cancelled'
                      ? 'Transaksi dibatalkan dan tidak berlaku lagi.'
                      : 'Pengajuan ini tidak dilanjutkan.'}
                </Alert>
                {rental.status === 'completed' ? (
                  <button
                    type="button"
                    className="btn btn-outline btn-block"
                    disabled={agreementBusy}
                    onClick={downloadAgreement}
                  >
                    {agreementBusy ? <span className="spinner" /> : <Icon name="download" />}
                    Unduh Surat Perjanjian
                  </button>
                ) : null}
                <Link to="/penyewaan" className="btn btn-ghost btn-block">
                  <Icon name="arrow_back" /> Kembali ke daftar
                </Link>
              </div>
            ) : null}

            <p className="hint mt-3">
              Pembayaran hanya dilakukan tunai saat COD. UNESA Rent tidak menampung transfer atau
              deposit.
            </p>
          </section>

          <section className="card card-pad">
            <h3 className="card-title">Informasi transaksi</h3>
            <dl className="kv mt-3">
              <dt>Status</dt>
              <dd><span className={`badge badge-${meta.tone}`}>{meta.label}</span></dd>
            </dl>
            <dl className="kv">
              <dt>Peranmu</dt>
              <dd>{isOwner ? 'Pemilik' : 'Penyewa'}</dd>
            </dl>
            <dl className="kv">
              <dt>Diajukan</dt>
              <dd>{formatDate(rental.created_at, 'short')}</dd>
            </dl>
            {rental.approved_at ? (
              <dl className="kv">
                <dt>Disetujui</dt>
                <dd>{formatDate(rental.approved_at, 'short')}</dd>
              </dl>
            ) : null}
            {rental.completed_at ? (
              <dl className="kv">
                <dt>Selesai</dt>
                <dd>{formatDate(rental.completed_at, 'short')}</dd>
              </dl>
            ) : null}
            <dl className="kv">
              <dt>ID transaksi</dt>
              <dd className="mono">#{rental.id.slice(0, 8).toUpperCase()}</dd>
            </dl>
          </section>

          <section className="card card-pad">
            <div className="party">
              <Avatar name={other?.name} size="sm" />
              <div className="party-body">
                <div className="party-name">{other?.name}</div>
                <div className="party-meta">{isOwner ? 'Penyewa' : 'Pemilik'}</div>
              </div>
            </div>
            <p className="hint mt-2">Butuh bantuan? Hubungi admin BEM FT lewat kanal kampus.</p>
          </section>
        </aside>
      </div>

      {rejectOpen ? (
        <Modal
          title="Tolak pengajuan sewa"
          subtitle={`${rental.renter?.name} · ${rental.listing?.title}`}
          onClose={() => setRejectOpen(false)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setRejectOpen(false)}>Batal</button>
              <button type="submit" form="rental-reason-form" className="btn btn-danger" disabled={modalBusy}>
                {modalBusy ? <span className="spinner" /> : <Icon name="cancel" />} Tolak Pengajuan
              </button>
            </>
          }
        >
          <form id="rental-reason-form" className="stack" onSubmit={submitReasonAction}>
            <div className="field">
              <label className="label" htmlFor="reject-reason">Alasan penolakan <span className="req">*</span></label>
              <textarea
                id="reject-reason"
                className="textarea"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: Barang sedang dipakai untuk praktikum minggu depan."
              />
            </div>
            {modalError ? <Alert tone="error" icon="error">{modalError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}

      {cancelOpen ? (
        <Modal
          title="Batalkan transaksi"
          subtitle={`${rental.listing?.title} · ${dateRange(rental.start_date, rental.end_date)}`}
          onClose={() => setCancelOpen(false)}
          footer={
            <>
              <button type="button" className="btn btn-ghost" onClick={() => setCancelOpen(false)}>Tutup</button>
              <button type="submit" form="rental-reason-form" className="btn btn-danger" disabled={modalBusy}>
                {modalBusy ? <span className="spinner" /> : <Icon name="block" />} Batalkan
              </button>
            </>
          }
        >
          <form id="rental-reason-form" className="stack" onSubmit={submitReasonAction}>
            <Alert tone="warning" icon="warning" title="Transaksi tidak dapat dikembalikan">
              Pihak lain akan menerima notifikasi pembatalan.
            </Alert>
            <div className="field">
              <label className="label" htmlFor="cancel-reason">
                Alasan pembatalan {isOwner ? <span className="req">*</span> : <span className="muted">(opsional)</span>}
              </label>
              <textarea
                id="cancel-reason"
                className="textarea"
                value={reason}
                onChange={(event) => setReason(event.target.value)}
                placeholder="Contoh: Barang rusak sebelum masa sewa dimulai."
              />
            </div>
            {modalError ? <Alert tone="error" icon="error">{modalError.message}</Alert> : null}
          </form>
        </Modal>
      ) : null}
    </div>
  )
}
