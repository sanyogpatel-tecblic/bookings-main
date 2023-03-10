package dbrepo

import (
	"context"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/sanyogpatel-tecblic/bookings/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (m *postgressDBRepo) AllUsers() bool {
	return true
}

var newId int

func (m *postgressDBRepo) InsertReservation(res models.Reservation) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	query := `insert into reservations(first_name,last_name,email,phone,start_date,end_date,room_id,created_at,updated_at)
			values ($1,$2,$3,$4,$5,$6,$7,$8,$9) returning id`

	err := m.DB.QueryRowContext(ctx, query,
		res.Firstname,
		res.Lastname,
		res.Email,
		res.Phone,
		res.StartDate,
		res.EndDate,
		res.RoomID,
		time.Now(),
		time.Now(),
	).Scan(&newId)
	if err != nil {
		return 0, err
	}
	return newId, nil
}

func (m *postgressDBRepo) InsertRoomRestriction(r models.RoomRestriction) error {

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `insert into room_restrictions(start_date,end_date,room_id,reservation_id,created_at,updated_at,restriction_id)
			values ($1,$2,$3,$4,$5,$6,$7)`

	_, err := m.DB.ExecContext(ctx, query,
		r.StartDate, r.EndDate, r.RoomID, r.ReservationID, time.Now(), time.Now(), r.RestrictionID,
	)
	if err != nil {
		return err
	}
	return nil
}
func (m *postgressDBRepo) SearchAvailabilityByDatesByRoomID(start, end time.Time, roomID int) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var numRows int

	query := `
		select
			count(id)
		from
			room_restrictions
		where
			room_id = $1
			and $2 < end_date and $3 > start_date;`

	row := m.DB.QueryRowContext(ctx, query, roomID, start, end)
	err := row.Scan(&numRows)
	if err != nil {
		return false, err
	}

	if numRows == 0 {
		return true, nil
	}
	return false, nil
}

func (m *postgressDBRepo) SearchAvailabilityForAllRooms(start, end time.Time) ([]models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var rooms []models.Room

	query := `select r.id,r.room_name from rooms r where r.id not in
				(select rr.room_id from room_restrictions rr
					where $1<rr.end_date and $2>rr.start_date)`

	rows, err := m.DB.QueryContext(ctx, query, start, end)
	if err != nil {
		return rooms, err
	}
	for rows.Next() {
		var room models.Room

		err := rows.Scan(
			&room.ID,
			&room.RoomName,
		)
		if err != nil {
			return rooms, err
		}
		rooms = append(rooms, room)
	}
	if err = rows.Err(); err != nil {
		log.Fatal("Error scanning rows", err)
	}
	return rooms, nil
}

func (m *postgressDBRepo) GetRoomByID(id int) (models.Room, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var room models.Room

	query := `select id,room_name from rooms where id = $1`

	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&room.ID,
		&room.RoomName,
	)
	if err != nil {
		return room, err
	}
	return room, nil
}
func (m *postgressDBRepo) GetUserByID(id int) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `select id,first_name,last_name,email,password,access_level,created_at,updated_at from users where id=$1`

	var user models.User
	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		user.Firstname,
		user.Lastname,
		user.Email,
		user.Password,
		user.AccessLevel,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return user, err
	}
	return user, nil
}
func (m *postgressDBRepo) UpdateUser(user models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `update users set first_name=$1,last_name=$2 ,email=$3 access_level=$4 updated_at=$5`
	_, err := m.DB.ExecContext(ctx, query,
		user.Firstname,
		user.Lastname,
		user.Email,
		user.AccessLevel,
		time.Now(),
	)
	if err != nil {
		return err
	}
	return nil

}

func (m *postgressDBRepo) Authenticate(email, testPassword string) (int, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedpassword string

	row := m.DB.QueryRowContext(ctx, `select id,password from users where email=$1`, email)

	err := row.Scan(&id, &hashedpassword)

	if err != nil {
		return id, "", err
	}
	err = bcrypt.CompareHashAndPassword([]byte(hashedpassword), []byte(testPassword))

	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, "", errors.New("incorrect password")
	} else if err != nil {
		return 0, "", err
	}
	return id, hashedpassword, nil
}
func (m *postgressDBRepo) AllReservations() ([]models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservations []models.Reservation

	query := `
		select r.id, r.first_name, r.last_name, r.email, r.phone, r.start_date, 
		r.end_date, r.room_id, r.created_at, r.updated_at,r.processed,
		rm.id, rm.room_name
		from reservations r
		left join rooms rm on (r.room_id = rm.id)
		order by r.start_date asc
`

	rows, err := m.DB.QueryContext(ctx, query)

	if err != nil {
		return reservations, err
	}
	defer rows.Close()
	for rows.Next() {
		var i models.Reservation

		err := rows.Scan(
			&i.ID,
			&i.Firstname,
			&i.Lastname,
			&i.Email,
			&i.Phone,
			&i.StartDate,
			&i.EndDate,
			&i.RoomID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Processed,
			&i.Room.ID,
			&i.Room.RoomName,
		)
		if err != nil {
			return reservations, err
		}
		reservations = append(reservations, i)
	}
	if err = rows.Err(); err != nil {
		return reservations, err
	}
	return reservations, nil
}

func (m *postgressDBRepo) AllNewReservations() ([]models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservations []models.Reservation

	query := `
		select r.id, r.first_name, r.last_name, r.email, r.phone, r.start_date, 
		r.end_date, r.room_id, r.created_at, r.updated_at, 
		rm.id, rm.room_name
		from reservations r
		left join rooms rm on (r.room_id = rm.id)
		where processed = 0
		order by r.start_date asc
`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return reservations, err
	}
	defer rows.Close()

	for rows.Next() {
		var i models.Reservation
		err := rows.Scan(
			&i.ID,
			&i.Firstname,
			&i.Lastname,
			&i.Email,
			&i.Phone,
			&i.StartDate,
			&i.EndDate,
			&i.RoomID,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Room.ID,
			&i.Room.RoomName,
		)

		if err != nil {
			return reservations, err
		}
		reservations = append(reservations, i)
	}
	if err = rows.Err(); err != nil {
		return reservations, err
	}
	return reservations, nil
}

func (m *postgressDBRepo) GetReservationsByID(id int) (models.Reservation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var reservation models.Reservation

	query := `select r.id,r.first_name,r.last_name,r.email,
			r.start_date,r.end_date,r.room_id,r.phone,r.processed,rm.id,rm.room_name from 
			reservations r
			left join rooms rm on(rm.id = r.room_id)where r.id=$1`

	row := m.DB.QueryRowContext(ctx, query, id)

	err := row.Scan(
		&reservation.ID,
		&reservation.Firstname,
		&reservation.Lastname,
		&reservation.Email,
		&reservation.StartDate,
		&reservation.EndDate,
		&reservation.RoomID,
		&reservation.Phone,
		&reservation.Processed,
		&reservation.Room.ID,
		&reservation.Room.RoomName,
	)
	if err != nil {
		return reservation, err
	}
	return reservation, nil
}
func (m *postgressDBRepo) UpdateReservation(user models.Reservation) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `update reservations set first_name=$1,last_name=$2 ,email=$3,phone=$4,updated_at=$5 where id=$6`
	_, err := m.DB.ExecContext(ctx, query,
		user.Firstname,
		user.Lastname,
		user.Email,
		user.Phone,
		time.Now(),
		user.ID,
	)
	if err != nil {
		return err
	}
	return nil
}
func (m *postgressDBRepo) DeleteReservation(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `delete from reservations where id=$1`

	_, err := m.DB.QueryContext(ctx, query, id)

	if err != nil {
		return err
	}
	return nil
}

func (m *postgressDBRepo) UpdateProcessedForReservation(id int, processed int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `update reservations set processed=$1 where id=$2`

	_, err := m.DB.QueryContext(ctx, query, processed, id)

	if err != nil {
		return err
	}
	return nil
}
